// Package signatures contains transport-independent signature primitives.
// Its exact Ed25519 identifier is part of Euxenion ticket profile v1.
package signatures

import (
 "crypto"
 "crypto/ecdsa"
 "crypto/ed25519"
 "crypto/elliptic"
 "crypto/rand"
 "crypto/sha256"
 "errors"
 "math/big"
)

type Algorithm string

const (
 Ed25519 Algorithm = "Ed25519"
 ES256 Algorithm = "ES256"
)

var (
 ErrAlgorithm = errors.New("unsupported signature algorithm")
 ErrKey = errors.New("signature key does not match algorithm")
 ErrSignature = errors.New("invalid signature")
)

func Supported(alg Algorithm) bool { return alg == Ed25519 || alg == ES256 }

// Sign signs exactly the supplied message bytes; the calling envelope owns
// its protected-header construction and signing input.
func Sign(alg Algorithm, key crypto.PrivateKey, message []byte) ([]byte,error) {
 switch alg {
 case Ed25519:
  k,ok:=key.(ed25519.PrivateKey)
  if !ok || len(k)!=ed25519.PrivateKeySize{return nil,ErrKey}
  return ed25519.Sign(k,message),nil
 case ES256:
  k,ok:=key.(*ecdsa.PrivateKey)
  if !ok || k==nil || k.Curve!=elliptic.P256() || k.D==nil ||
    k.D.Sign()<=0 || k.D.Cmp(k.Params().N)>=0{return nil,ErrKey}
  sum:=sha256.Sum256(message)
  r,s,err:=ecdsa.Sign(rand.Reader,k,sum[:])
  if err!=nil {return nil,ErrSignature}
  sig:=make([]byte,64)
  r.FillBytes(sig[:32]);s.FillBytes(sig[32:])
  return sig,nil
 default:return nil,ErrAlgorithm
 }
}

// Verify rejects algorithm confusion, non-P256 curves, ASN.1 DER ES256
// signatures and malformed/range-invalid R||S values. It does not grant access.
func Verify(alg Algorithm,key crypto.PublicKey,message,signature []byte)error{
 switch alg {
 case Ed25519:
  k,ok:=key.(ed25519.PublicKey)
  if !ok || len(k)!=ed25519.PublicKeySize{return ErrKey}
  if len(signature)!=ed25519.SignatureSize || !ed25519.Verify(k,message,signature){return ErrSignature}
  return nil
 case ES256:
  k,ok:=key.(*ecdsa.PublicKey)
  if !ok || k==nil || k.Curve!=elliptic.P256() || k.X==nil || k.Y==nil ||
    !k.Curve.IsOnCurve(k.X,k.Y){return ErrKey}
  if len(signature)!=64{return ErrSignature}
  r:=new(big.Int).SetBytes(signature[:32])
  s:=new(big.Int).SetBytes(signature[32:])
  n:=k.Params().N
  if r.Sign()<=0 || s.Sign()<=0 || r.Cmp(n)>=0 || s.Cmp(n)>=0{return ErrSignature}
  hash:=sha256.Sum256(message)
  if !ecdsa.Verify(k,hash[:],r,s){return ErrSignature}
  return nil
 default:return ErrAlgorithm
 }
}
