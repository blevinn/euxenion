package trust

import (
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "testing"

 "github.com/blevinn/euxenion/internal/keys"
 "github.com/blevinn/euxenion/internal/signatures"
)

func TestIssuerTrust(t *testing.T){
 d:=t.TempDir()
 p,e:=keys.Generate(signatures.Ed25519);if e!=nil{t.Fatal(e)}
 pub,e:=keys.MarshalPublic(p.Public);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(filepath.Join(d,"issuer.pem"),pub,0644);e!=nil{t.Fatal(e)}
 cfg:=Config{Version:1,Issuers:[]IssuerConfig{{ID:"internal",Keys:[]KeyConfig{{ID:p.ID,Algorithm:signatures.Ed25519,PublicFile:"issuer.pem"}}},{ID:"external",Keys:[]KeyConfig{{ID:p.ID,Algorithm:signatures.Ed25519,PublicFile:"issuer.pem"}}}}}
 b,_:=json.Marshal(cfg);r,e:=Load(b,d);if e!=nil{t.Fatal(e)}
 for _,issuer:=range []string{"internal","external"}{
  if _,e=r.Lookup(issuer,p.ID,signatures.Ed25519);e!=nil{t.Fatal(e)}
 }
 for _,tt:=range []struct{issuer,kid string;alg signatures.Algorithm}{{"attacker",p.ID,signatures.Ed25519},{"internal","unknown",signatures.Ed25519},{"external",p.ID,signatures.ES256}}{
  if _,e=r.Lookup(tt.issuer,tt.kid,tt.alg);!errors.Is(e,ErrTrust){t.Fatalf("accepted bad lookup: %v",e)}
 }
 for _,malformed:=range []string{
  `{"v":1,"v":1,"issuers":[]}`,
  `{"v":1,"unknown":true,"issuers":[]}`,
  `{"v":1,"issuers":[{"id":"dup","keys":[]},{"id":"dup","keys":[]}]}`,
 }{
  if _,e:=Load([]byte(malformed),d);!errors.Is(e,ErrTrust){t.Fatalf("accepted config: %s",malformed)}
 }
 cfg.Issuers=append(cfg.Issuers,cfg.Issuers[0]);b,_=json.Marshal(cfg)
 if _,e=Load(b,d);!errors.Is(e,ErrTrust){t.Fatal(e)}
 cfg.Issuers=cfg.Issuers[:1];cfg.Issuers[0].Keys[0].Algorithm=signatures.ES256
 b,_=json.Marshal(cfg)
 if _,e=Load(b,d);!errors.Is(e,ErrTrust){t.Fatal(e)}
}
func TestPathAndConfigSafety(t *testing.T){
 d:=t.TempDir();cfg:=[]byte(`{"v":1,"issuers":[{"id":"test","keys":[{"kid":"x","alg":"Ed25519","public_key":"../escape.pem"}]}]}`)
 if _,e:=Load(cfg,d);!errors.Is(e,ErrTrust){t.Fatal(e)}
 p:=filepath.Join(d,"trust.json")
 if e:=os.WriteFile(p,cfg,0666);e!=nil{t.Fatal(e)}
 if e:=os.Chmod(p,0666);e!=nil{t.Fatal(e)}
 if _,e:=LoadFile(p);!errors.Is(e,ErrTrust){t.Fatal(e)}
}
