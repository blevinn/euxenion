package keys

import (
 "bytes"
 "crypto/ed25519"
 "errors"
 "os"
 "path/filepath"
 "testing"

 "github.com/blevinn/euxenion/internal/signatures"
)

func TestKeysAndEncryption(t *testing.T){
 for _,alg:=range []signatures.Algorithm{signatures.Ed25519,signatures.ES256}{
  pair,e:=Generate(alg);if e!=nil{t.Fatal(e)}
  pub,e:=MarshalPublic(pair.Public);if e!=nil{t.Fatal(e)}
  pubKey,a,id,e:=ParsePublic(pub);if e!=nil||a!=alg||id!=pair.ID{t.Fatalf("public mismatch: %v",e)}
  signature,e:=signatures.Sign(alg,pair.Private,[]byte("test"));if e!=nil{t.Fatal(e)}
  if e=signatures.Verify(alg,pubKey,[]byte("test"),signature);e!=nil{t.Fatal(e)}
  password:=[]byte("test passphrase for fixture")
  encrypted,e:=EncryptPrivate(pair,password);if e!=nil{t.Fatal(e)}
  if bytes.Contains(encrypted,[]byte("BEGIN PRIVATE KEY")){t.Fatal("unencrypted key")}
  recovered,e:=DecryptPrivate(encrypted,password);if e!=nil||recovered.ID!=id{t.Fatalf("decrypt: %v",e)}
  _,e=DecryptPrivate(encrypted,[]byte("an incorrect passphrase"));if !errors.Is(e,ErrDecrypt){t.Fatalf("wrong password: %v",e)}
  encrypted[len(encrypted)/2]^=1
  if _,e=DecryptPrivate(encrypted,password);e==nil{t.Fatal("corruption accepted")}
 }
}
func TestFileSafety(t *testing.T){
 dir:=t.TempDir();p:=filepath.Join(dir,"private.key")
 if e:=WriteExclusive(p,[]byte("test"),true);e!=nil{t.Fatal(e)}
 if e:=WriteExclusive(p,[]byte("overwrite"),true);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
 info,_:=os.Stat(p);if info.Mode().Perm()!=0600{t.Fatalf("permissions %v",info.Mode())}
 if _,e:=ReadEncrypted(p);e!=nil{t.Fatal(e)}
 unsafe:=filepath.Join(dir,"unsafe.key")
 if e:=os.WriteFile(unsafe,[]byte("secret"),0644);e!=nil{t.Fatal(e)}
 if _,e:=ReadEncrypted(unsafe);!errors.Is(e,ErrUnsafe){t.Fatal(e)}
 link:=filepath.Join(dir,"link")
 if e:=os.Symlink(p,link);e==nil{if _,e:=ReadEncrypted(link);!errors.Is(e,ErrUnsafe){t.Fatal(e)}}
 if _,_,_,e:=ParsePublic([]byte("bad"));!errors.Is(e,ErrInvalid){t.Fatal(e)}
 if _,e:=AlgorithmOf(ed25519.PublicKey(make([]byte,20)));!errors.Is(e,ErrInvalid){t.Fatal(e)}
}
