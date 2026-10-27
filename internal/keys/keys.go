// Package keys provides a minimal versioned encrypted private-key file and
// interoperable PKIX PEM public keys. It has no access-control or MCP logic.
package keys

import (
 "bytes"
 "crypto"
 "crypto/aes"
 "crypto/cipher"
 "crypto/ecdsa"
 "crypto/ed25519"
 "crypto/elliptic"
 "crypto/hmac"
 "crypto/rand"
 "crypto/sha256"
 "crypto/x509"
 "encoding/base64"
 "encoding/json"
 "encoding/pem"
 "errors"
 "io"
 "os"
 "path/filepath"
 "syscall"

 "github.com/blevinn/euxenion/internal/signatures"
)

var (
 ErrInvalid = errors.New("invalid key material or encrypted key format")
 ErrUnsafe = errors.New("unsafe key file or passphrase source")
 ErrDecrypt = errors.New("key authentication failed")
)

const iterations = 600000
type Pair struct {
 Algorithm signatures.Algorithm
 ID string
 Private crypto.PrivateKey
 Public crypto.PublicKey
}
type Envelope struct {
 Version int `json:"v"`
 Algorithm signatures.Algorithm `json:"alg"`
 ID string `json:"kid"`
 KDF string `json:"kdf"`
 Iterations int `json:"iterations"`
 Salt string `json:"salt"`
 Cipher string `json:"cipher"`
 Nonce string `json:"nonce"`
 Ciphertext string `json:"ciphertext"`
}

func Generate(alg signatures.Algorithm) (Pair,error){
 var private crypto.PrivateKey
 var public crypto.PublicKey
 switch alg {
 case signatures.Ed25519:
  pub,priv,e:=ed25519.GenerateKey(rand.Reader);if e!=nil{return Pair{},ErrInvalid}
  private,public=priv,pub
 case signatures.ES256:
  priv,e:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader);if e!=nil{return Pair{},ErrInvalid}
  private,public=priv,&priv.PublicKey
 default:return Pair{},signatures.ErrAlgorithm
 }
 id,e:=KeyID(public);if e!=nil{return Pair{},e}
 return Pair{Algorithm:alg,ID:id,Private:private,Public:public},nil
}

func AlgorithmOf(pub crypto.PublicKey)(signatures.Algorithm,error){
 switch k:=pub.(type){
 case ed25519.PublicKey:
  if len(k)==ed25519.PublicKeySize{return signatures.Ed25519,nil}
 case *ecdsa.PublicKey:
  if k!=nil && k.Curve==elliptic.P256() && k.X!=nil && k.Y!=nil && k.Curve.IsOnCurve(k.X,k.Y){return signatures.ES256,nil}
 }
 return "",ErrInvalid
}

func KeyID(public crypto.PublicKey)(string,error){
 if _,e:=AlgorithmOf(public);e!=nil{return "",e}
 b,e:=x509.MarshalPKIXPublicKey(public);if e!=nil{return "",ErrInvalid}
 hash:=sha256.Sum256(b)
 return base64.RawURLEncoding.EncodeToString(hash[:16]),nil
}

func MarshalPublic(public crypto.PublicKey)([]byte,error){
 if _,e:=AlgorithmOf(public);e!=nil{return nil,e}
 b,e:=x509.MarshalPKIXPublicKey(public);if e!=nil{return nil,ErrInvalid}
 return pem.EncodeToMemory(&pem.Block{Type:"PUBLIC KEY",Bytes:b}),nil
}
func ParsePublic(data []byte)(crypto.PublicKey,signatures.Algorithm,string,error){
 if len(data)==0 || len(data)>8192{return nil,"","",ErrInvalid}
 block,rest:=pem.Decode(data)
 if block==nil || block.Type!="PUBLIC KEY" || len(bytes.TrimSpace(rest))!=0{return nil,"","",ErrInvalid}
 key,e:=x509.ParsePKIXPublicKey(block.Bytes);if e!=nil{return nil,"","",ErrInvalid}
 alg,e:=AlgorithmOf(key);if e!=nil{return nil,"","",e}
 id,e:=KeyID(key);return key,alg,id,e
}

func privatePublic(key crypto.PrivateKey)(crypto.PublicKey,error){
 switch k:=key.(type) {
 case ed25519.PrivateKey:
  if len(k)!=ed25519.PrivateKeySize{return nil,ErrInvalid}
  return k.Public(),nil
 case *ecdsa.PrivateKey:
  if k==nil || k.Curve!=elliptic.P256() || k.D==nil || k.D.Sign()<=0 || k.D.Cmp(k.Params().N)>=0{return nil,ErrInvalid}
  return &k.PublicKey,nil
 default:return nil,ErrInvalid
 }
}

// PBKDF2-HMAC-SHA256, one 32-byte block. Storage only; not a protocol KDF.
func derive(password,salt []byte,rounds int)[]byte{
 var count=[4]byte{0,0,0,1}
 mac:=hmac.New(sha256.New,password);mac.Write(salt);mac.Write(count[:])
 u:=mac.Sum(nil);out:=append([]byte(nil),u...)
 for i:=1;i<rounds;i++{
  mac=hmac.New(sha256.New,password);mac.Write(u);u=mac.Sum(u[:0])
  for j:=range out{out[j]^=u[j]}
 }
 return out
}
func aad(alg signatures.Algorithm,id string)[]byte{return []byte("euxenion.encrypted-pkcs8.v1:"+string(alg)+":"+id)}
func checkPassword(password []byte)error{
 if len(password)<12 || len(password)>1024{return ErrUnsafe}
 return nil
}

// EncryptPrivate encodes PKCS#8 DER with PBKDF2-SHA256/AES-256-GCM.
// The caller must provide a securely acquired passphrase, never a CLI argument.
func EncryptPrivate(pair Pair,password []byte)([]byte,error){
 if e:=checkPassword(password);e!=nil{return nil,e}
 pub,e:=privatePublic(pair.Private);if e!=nil{return nil,e}
 alg,e:=AlgorithmOf(pub);if e!=nil || alg!=pair.Algorithm{return nil,ErrInvalid}
 id,e:=KeyID(pub);if e!=nil || id!=pair.ID{return nil,ErrInvalid}
 der,e:=x509.MarshalPKCS8PrivateKey(pair.Private);if e!=nil{return nil,ErrInvalid}
 defer clear(der)
 salt:=make([]byte,16);nonce:=make([]byte,12)
 if _,e=rand.Read(salt);e!=nil{return nil,ErrInvalid}
 if _,e=rand.Read(nonce);e!=nil{return nil,ErrInvalid}
 derived:=derive(password,salt,iterations);defer clear(derived)
 block,e:=aes.NewCipher(derived);if e!=nil{return nil,ErrInvalid}
 gcm,e:=cipher.NewGCM(block);if e!=nil{return nil,ErrInvalid}
 ct:=gcm.Seal(nil,nonce,der,aad(alg,id))
 env:=Envelope{Version:1,Algorithm:alg,ID:id,KDF:"PBKDF2-HMAC-SHA256",Iterations:iterations,
  Salt:base64.RawURLEncoding.EncodeToString(salt),Cipher:"AES-256-GCM",
  Nonce:base64.RawURLEncoding.EncodeToString(nonce),Ciphertext:base64.RawURLEncoding.EncodeToString(ct)}
 out,e:=json.Marshal(env);if e!=nil{return nil,ErrInvalid}
 return append(out,'\n'),nil
}

func decodeEnvelope(data []byte)(Envelope,error){
 var env Envelope
 if len(data)<100 || len(data)>16384{return env,ErrInvalid}
 dec:=json.NewDecoder(bytes.NewReader(data));dec.DisallowUnknownFields()
 if e:=dec.Decode(&env);e!=nil{return env,ErrInvalid}
 var extra any
 if e:=dec.Decode(&extra);e!=io.EOF{return env,ErrInvalid}
 if env.Version!=1 || !signatures.Supported(env.Algorithm) ||
  env.ID=="" || len(env.ID)>64 || env.KDF!="PBKDF2-HMAC-SHA256" ||
  env.Cipher!="AES-256-GCM" || env.Iterations<600000 || env.Iterations>2000000{return env,ErrInvalid}
 for _,c:=range env.ID{if !((c>='A'&&c<='Z')||(c>='a'&&c<='z')||(c>='0'&&c<='9')||c=='_'||c=='-'){return env,ErrInvalid}}
 salt,e1:=base64.RawURLEncoding.DecodeString(env.Salt)
 nonce,e2:=base64.RawURLEncoding.DecodeString(env.Nonce)
 ct,e3:=base64.RawURLEncoding.DecodeString(env.Ciphertext)
 if e1!=nil||e2!=nil||e3!=nil||len(salt)!=16||len(nonce)!=12||len(ct)<17||len(ct)>8192{return env,ErrInvalid}
 return env,nil
}

// InspectEncrypted returns only non-secret algorithm and key-ID metadata.
func InspectEncrypted(data []byte)(signatures.Algorithm,string,error){
 env,e:=decodeEnvelope(data);if e!=nil{return "","",e}
 return env.Algorithm,env.ID,nil
}
func DecryptPrivate(data,password []byte)(Pair,error){
 env,e:=decodeEnvelope(data);if e!=nil{return Pair{},e}
 if e=checkPassword(password);e!=nil{return Pair{},e}
 salt,_:=base64.RawURLEncoding.DecodeString(env.Salt)
 nonce,_:=base64.RawURLEncoding.DecodeString(env.Nonce)
 ct,_:=base64.RawURLEncoding.DecodeString(env.Ciphertext)
 derived:=derive(password,salt,env.Iterations);defer clear(derived)
 block,e:=aes.NewCipher(derived);if e!=nil{return Pair{},ErrInvalid}
 gcm,e:=cipher.NewGCM(block);if e!=nil{return Pair{},ErrInvalid}
 plaintext,e:=gcm.Open(nil,nonce,ct,aad(env.Algorithm,env.ID))
 if e!=nil{return Pair{},ErrDecrypt}
 defer clear(plaintext)
 private,e:=x509.ParsePKCS8PrivateKey(plaintext);if e!=nil{return Pair{},ErrInvalid}
 public,e:=privatePublic(private);if e!=nil{return Pair{},e}
 alg,e:=AlgorithmOf(public);if e!=nil || alg!=env.Algorithm{return Pair{},ErrInvalid}
 id,e:=KeyID(public);if e!=nil || id!=env.ID{return Pair{},ErrInvalid}
 return Pair{Algorithm:alg,ID:id,Private:private,Public:public},nil
}

func readSafe(path string, secret bool)([]byte,error){
 info,e:=os.Lstat(path)
 if e!=nil{return nil,ErrUnsafe}
 if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink!=0 || (secret && info.Mode().Perm()&0077!=0){return nil,ErrUnsafe}
 f,e:=os.Open(path);if e!=nil{return nil,ErrUnsafe}
 defer f.Close()
 b,e:=io.ReadAll(io.LimitReader(f,16385));if e!=nil || len(b)>16384{return nil,ErrUnsafe}
 return b,nil
}
func ReadEncrypted(path string)([]byte,error){return readSafe(path,true)}
func ReadPassphrase(path string)([]byte,error){
 b,e:=readSafe(path,true);if e!=nil{return nil,e}
 b=bytes.TrimSuffix(b,[]byte("\n"))
 b=bytes.TrimSuffix(b,[]byte("\r"))
 if e=checkPassword(b);e!=nil{return nil,e}
 return b,nil
}
// WriteExclusive never follows a symlink or overwrites an existing file.
// It fails for any non-regular existing destination and uses restrictive mode.
func WriteExclusive(path string,contents []byte,private bool)error{
 mode:=uint32(0644);if private{mode=0600}
 parent:=filepath.Dir(path)
 fi,e:=os.Lstat(parent);if e!=nil||!fi.IsDir()||fi.Mode()&os.ModeSymlink!=0{return ErrUnsafe}
 fd,e:=syscall.Open(path,syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW,mode)
 if e!=nil{return ErrUnsafe}
 f:=os.NewFile(uintptr(fd),path)
 _,e=f.Write(contents);closeErr:=f.Close()
 if e!=nil||closeErr!=nil{_ = os.Remove(path);return ErrUnsafe}
 return nil
}
