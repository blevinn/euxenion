package keys

import (
 "bytes"
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/blevinn/euxenion/internal/signatures"
)

func TestEnvelopeAdversarial(t *testing.T){
 pair,e:=Generate(signatures.Ed25519);if e!=nil{t.Fatal(e)}
 pw:=[]byte("this is a high entropy test-only passphrase")
 source,e:=EncryptPrivate(pair,pw);if e!=nil{t.Fatal(e)}
 var orig Envelope
 if e=json.Unmarshal(source,&orig);e!=nil{t.Fatal(e)}
 cases:=[]struct{name string; mutate func(*Envelope)}{
  {"weak kdf",func(v *Envelope){v.Iterations=1}},
  {"excess kdf",func(v *Envelope){v.Iterations=2000001}},
  {"unknown algorithm",func(v *Envelope){v.Algorithm="none"}},
  {"mismatch algorithm",func(v *Envelope){v.Algorithm=signatures.ES256}},
  {"wrong kid",func(v *Envelope){v.ID=strings.Repeat("A",22)}},
  {"empty salt",func(v *Envelope){v.Salt=""}},
  {"short nonce",func(v *Envelope){v.Nonce="AA"}},
  {"empty cipher",func(v *Envelope){v.Cipher=""}},
 }
 for _,tt:=range cases{t.Run(tt.name,func(t *testing.T){
  v:=orig;tt.mutate(&v);data,_:=json.Marshal(v)
  if _,e:=DecryptPrivate(data,pw);e==nil{t.Fatal("accepted tampered envelope")}
 })}
 duplicate:=bytes.Replace(source,[]byte(`"v":1,`),[]byte(`"v":1,"v":1,`),1)
 if _,e:=InspectEncrypted(duplicate);!errors.Is(e,ErrInvalid){t.Fatalf("duplicate accepted: %v",e)}
 malformed:=bytes.Replace(source,[]byte(`"kdf":"PBKDF2-HMAC-SHA256",`),[]byte(`"kdf":"PBKDF2-HMAC-SHA256","unknown":1,`),1)
 if _,e:=InspectEncrypted(malformed);!errors.Is(e,ErrInvalid){t.Fatalf("unknown accepted: %v",e)}
 if _,e:=DecryptPrivate(source,[]byte("another incorrect test password"));!errors.Is(e,ErrDecrypt){t.Fatal(e)}
 if _,e:=DecryptPrivate(source,nil);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
}
func TestSensitiveFilePaths(t *testing.T){
 dir:=t.TempDir();secret:=filepath.Join(dir,"secret")
 if e:=os.WriteFile(secret,[]byte("test-only-passphrase-12345\n"),0600);e!=nil{t.Fatal(e)}
 pass,e:=ReadPassphrase(secret);if e!=nil||string(pass)!="test-only-passphrase-12345"{t.Fatalf("pass error %v",e)}
 link:=filepath.Join(dir,"symlink")
 if e:=os.Symlink(secret,link);e==nil{
  if _,e:=ReadPassphrase(link);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
  if e:=WriteExclusive(link,[]byte("bad"),true);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
 }
 if e:=os.Chmod(secret,0644);e!=nil{t.Fatal(e)}
 if _,e:=ReadPassphrase(secret);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
}
