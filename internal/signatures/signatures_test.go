package signatures

import (
 "crypto/ecdsa"
 "crypto/ed25519"
 "crypto/elliptic"
 "crypto/rand"
 "encoding/hex"
 "errors"
 "testing"
)

func TestSignVerify(t *testing.T){
 edPub,edPriv,e:=ed25519.GenerateKey(rand.Reader);if e!=nil{t.Fatal(e)}
 ecPriv,e:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader);if e!=nil{t.Fatal(e)}
 msg:=[]byte("explicit binding")
 for _,tt:=range []struct{alg Algorithm; priv any; pub any}{{Ed25519,edPriv,edPub},{ES256,ecPriv,&ecPriv.PublicKey}}{
  sig,e:=Sign(tt.alg,tt.priv,msg);if e!=nil{t.Fatal(e)}
  if tt.alg==ES256 && len(sig)!=64{t.Fatal("not JWS raw signature")}
  if e:=Verify(tt.alg,tt.pub,msg,sig);e!=nil{t.Fatal(e)}
  if e:=Verify(tt.alg,tt.pub,[]byte("tampered"),sig);!errors.Is(e,ErrSignature){t.Fatalf("tamper accepted: %v",e)}
  sig[0]^=0xff
  if e:=Verify(tt.alg,tt.pub,msg,sig);!errors.Is(e,ErrSignature){t.Fatalf("mutation accepted: %v",e)}
 }
 if _,e:=Sign("EdDSA",edPriv,msg);!errors.Is(e,ErrAlgorithm){t.Fatal(e)}
 if _,e:=Sign("none",edPriv,msg);!errors.Is(e,ErrAlgorithm){t.Fatal(e)}
 if _,e:=Sign(ES256,edPriv,msg);!errors.Is(e,ErrKey){t.Fatal(e)}
 if e:=Verify(Ed25519,&ecPriv.PublicKey,msg,make([]byte,64));!errors.Is(e,ErrKey){t.Fatal(e)}
 if e:=Verify(ES256,edPub,msg,make([]byte,64));!errors.Is(e,ErrKey){t.Fatal(e)}
 if e:=Verify(ES256,&ecPriv.PublicKey,msg,[]byte{0x30,0x06,0x02,0x01,0x01});!errors.Is(e,ErrSignature){t.Fatal(e)}
 if e:=Verify(ES256,&ecPriv.PublicKey,msg,make([]byte,64));!errors.Is(e,ErrSignature){t.Fatal(e)}
 p384,e:=ecdsa.GenerateKey(elliptic.P384(),rand.Reader);if e!=nil{t.Fatal(e)}
 if _,e:=Sign(ES256,p384,msg);!errors.Is(e,ErrKey){t.Fatal(e)}
 if e:=Verify(ES256,&p384.PublicKey,msg,make([]byte,64));!errors.Is(e,ErrKey){t.Fatal(e)}
}
func TestExternalRFC8032Vector(t *testing.T){
 // RFC 8032 test vector 1, empty message, generated outside this library.
 pub,_:=hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
 sig,_:=hex.DecodeString("e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
 if e:=Verify(Ed25519,ed25519.PublicKey(pub),nil,sig);e!=nil{t.Fatal(e)}
}
