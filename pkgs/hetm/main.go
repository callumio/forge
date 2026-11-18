package main

import (
	"cmp"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const issuer = "https://token.actions.githubusercontent.com"

func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s required", k)
	}
	return v
}

func try[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// trailing * means prefix match
func match(p, s string) bool {
	pre, ok := strings.CutSuffix(p, "*")
	return ok && strings.HasPrefix(s, pre) || p == s
}

func main() {
	var (
		aud   = must("AUDIENCE")
		uid   = try(strconv.ParseUint(cmp.Or(os.Getenv("HS_USER_ID"), "0"), 10, 64)) // 0 = tags-only key
		allow = strings.Split(must("ALLOW"), ",")
		tags  = strings.Split(cmp.Or(os.Getenv("TAGS"), "tag:ci"), ",")
		ttl   = try(time.ParseDuration(cmp.Or(os.Getenv("TTL"), "5m")))
		sock  = cmp.Or(os.Getenv("HS_SOCK"), "/run/headscale/headscale.sock")
		addr  = net.JoinHostPort(cmp.Or(os.Getenv("BIND"), "127.0.0.1"), cmp.Or(os.Getenv("PORT"), "8081"))
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	prov := try(oidc.NewProvider(ctx, issuer))
	cancel()
	ver := prov.Verifier(&oidc.Config{ClientID: aud})

	conn := try(grpc.NewClient("unix://"+sock, grpc.WithTransportCredentials(insecure.NewCredentials())))
	defer conn.Close()
	hs := v1.NewHeadscaleServiceClient(conn)

	mint := func(ctx context.Context) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		r, err := hs.CreatePreAuthKey(ctx, &v1.CreatePreAuthKeyRequest{
			User:       uid,
			Ephemeral:  true,
			Expiration: timestamppb.New(time.Now().Add(ttl)),
			AclTags:    tags,
		})
		if err != nil {
			return "", err
		}
		return r.PreAuthKey.Key, nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			http.Error(w, "missing token", 401)
			return
		}
		tok, err := ver.Verify(r.Context(), raw)
		if err != nil {
			http.Error(w, "invalid token", 401)
			return
		}
		if !slices.ContainsFunc(allow, func(p string) bool { return match(p, tok.Subject) }) {
			log.Printf("denied sub=%s", tok.Subject)
			http.Error(w, "forbidden", 403)
			return
		}
		key, err := mint(r.Context())
		if err != nil {
			log.Printf("mint sub=%s: %v", tok.Subject, err)
			http.Error(w, "upstream error", 502)
			return
		}
		log.Printf("minted sub=%s", tok.Subject)
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]string{"key": key})
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
