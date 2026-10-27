package tickets

import (
 "crypto/rand"
 "encoding/base64"
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"

 "github.com/blevinn/euxenion/internal/keys"
 "github.com/blevinn/euxenion/internal/signatures"
 "github.com/blevinn/euxenion/internal/trust"
)

func random16(t *testing.T)string{t.Helper();b:=make([]byte,16);if _,e:=rand.Read(b);e!=nil{t.Fatal(e)};return base64.RawURLEncoding.EncodeToString(b)}
func setup(t *testing.T,alg signatures.Algorithm)(keys.Pair,*trust.Registry,Payload,time.Time){
 t.Helper();pair,e:=keys.Generate(alg);if e!=nil{t.Fatal(e)}
 d:=t.TempDir();pem,e:=keys.MarshalPublic(pair.Public);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(filepath.Join(d,"pub.pem"),pem,0644);e!=nil{t.Fatal(e)}
 config:=trust.Config{Version:1,Issuers:[]trust.IssuerConfig{{ID:"issuer.external",Keys:[]trust.KeyConfig{{ID:pair.ID,Algorithm:alg,PublicFile:"pub.pem"}}}}}
 configBytes,_:=json.Marshal(config);reg,e:=trust.Load(configBytes,d);if e!=nil{t.Fatal(e)}
 now:=time.Unix(1793000000,0)
 p:=Payload{Version:1,Issuer:"issuer.external",Audience:"proxy.prod",ID:random16(t),Issued:now.Unix()-30,Expires:now.Unix()+3600,
  Invocation:Invocation{Version:1,SHA256:base64.RawURLEncoding.EncodeToString(make([]byte,32))},
  Claims:map[string]string{"approver":"human"}}
 return pair,reg,p,now
}
func TestTicketRoundTrip(t *testing.T){
 for _,alg:=range []signatures.Algorithm{signatures.Ed25519,signatures.ES256}{
  t.Run(string(alg),func(t *testing.T){
   pair,reg,p,now:=setup(t,alg)
   token,e:=Issue(p,pair);if e!=nil{t.Fatal(e)}
   ev,e:=Verify(token,p.Audience,p.Invocation.SHA256,reg,now)
   if e!=nil || ev.Issuer!=p.Issuer || ev.KeyID!=pair.ID || ev.Claims["approver"]!="human"{t.Fatalf("verify: %#v %v",ev,e)}
   ev.Claims["approver"]="mutated"
   ev2,e:=Verify(token,p.Audience,p.Invocation.SHA256,reg,now)
   if e!=nil||ev2.Claims["approver"]!="human"{t.Fatal("evidence alias or verification failure",e)}
   for _,bad:=range []struct{name,ticket,aud,digest string; clock time.Time;want error}{
    {"wrong audience",token,"proxy.other",p.Invocation.SHA256,now,ErrInvalidBinding},
    {"wrong binding",token,p.Audience,base64.RawURLEncoding.EncodeToString(make([]byte,32-1)),now,ErrInvalidBinding},
    {"expired",token,p.Audience,p.Invocation.SHA256,time.Unix(p.Expires,0),ErrInvalidTime},
    {"invalid signature",token[:len(token)-1]+"A",p.Audience,p.Invocation.SHA256,now,ErrInvalidSignature},
   }{
    t.Run(bad.name,func(t *testing.T){_,e:=Verify(bad.ticket,bad.aud,bad.digest,reg,bad.clock);if !errors.Is(e,bad.want){t.Fatalf("wanted %v got %v",bad.want,e)}})
   }
   if _,e:=Verify(token,p.Audience,p.Invocation.SHA256,nil,now);!errors.Is(e,ErrUntrusted){t.Fatal(e)}
  })
 }
}
func TestMalformedAndTime(t *testing.T){
 pair,reg,p,now:=setup(t,signatures.Ed25519)
 p.Issued=now.Unix()+31;p.Expires=p.Issued+10
 token,e:=Issue(p,pair);if e!=nil{t.Fatal(e)}
 if _,e:=Verify(token,p.Audience,p.Invocation.SHA256,reg,now);!errors.Is(e,ErrInvalidTime){t.Fatal(e)}
 p.Issued=now.Unix()-10;p.Expires=p.Issued+86401
 if _,e=Issue(p,pair);!errors.Is(e,ErrInvalidTime){t.Fatal(e)}
 p.Expires=p.Issued+10;p.ID="not-random"
 if _,e=Issue(p,pair);!errors.Is(e,ErrMalformed){t.Fatal(e)}
 p.ID=random16(t);p.Claims=map[string]string{"too_many_?":"x"}
 if _,e=Issue(p,pair);!errors.Is(e,ErrMalformed){t.Fatal(e)}
 if _,e=Verify(strings.Repeat("A",MaxToken+1),p.Audience,p.Invocation.SHA256,reg,now);!errors.Is(e,ErrMalformed){t.Fatal(e)}
}
