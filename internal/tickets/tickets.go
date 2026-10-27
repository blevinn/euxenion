// Package tickets implements strict version-one compact JWS cryptographic
// evidence. Verified tickets are NEVER authorization to execute tools.
package tickets

import (
 "crypto/hmac"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "regexp"
 "strings"
 "time"

 "github.com/blevinn/euxenion/internal/keys"
 "github.com/blevinn/euxenion/internal/signatures"
 "github.com/blevinn/euxenion/internal/strictjson"
 "github.com/blevinn/euxenion/internal/trust"
)

const Type="euxenion.ticket.v1"
const MaxToken=8192

var (
 ErrMalformed=errors.New("malformed authorization evidence")
 ErrUnsupported=errors.New("unsupported authorization evidence")
 ErrUntrusted=errors.New("untrusted authorization evidence")
 ErrInvalidSignature=errors.New("invalid evidence signature")
 ErrInvalidBinding=errors.New("invalid evidence binding")
 ErrInvalidTime=errors.New("invalid evidence validity period")
 ErrInternal=errors.New("evidence verification unavailable")
)

type Header struct {
 Type string `json:"typ"`
 Algorithm signatures.Algorithm `json:"alg"`
 KeyID string `json:"kid"`
}
type Invocation struct{
 Version int `json:"v"`
 SHA256 string `json:"sha256"`
}
type Payload struct{
 Version int `json:"v"`
 Issuer string `json:"iss"`
 Audience string `json:"aud"`
 ID string `json:"jti"`
 Issued int64 `json:"iat"`
 NotBefore *int64 `json:"nbf,omitempty"`
 Expires int64 `json:"exp"`
 Invocation Invocation `json:"inv"`
 Claims map[string]string `json:"claims"`
 ChallengeID string `json:"challenge_id,omitempty"`
}
type Evidence struct{
 Issuer string
 KeyID string
 Audience string
 TicketID string
 InvocationSHA256 string
 Claims map[string]string
 ChallengeID string
 Expires time.Time
}
var issuer=regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)
var key=regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
var claimKey=regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

func decoded(s string,size int)bool{
 if len(s)!=base64.RawURLEncoding.EncodedLen(size){return false}
 b,e:=base64.RawURLEncoding.DecodeString(s)
 return e==nil && len(b)==size && base64.RawURLEncoding.EncodeToString(b)==s
}
func shape(p Payload)error{
 if p.Version!=1 || !issuer.MatchString(p.Issuer) || !issuer.MatchString(p.Audience) ||
  !decoded(p.ID,16) || p.Invocation.Version!=1 || !decoded(p.Invocation.SHA256,32) ||
  len(p.Claims)>32 || p.Claims==nil{return ErrMalformed}
 for k,v:=range p.Claims{
  if !claimKey.MatchString(k) || len(v)==0 || len([]byte(v))>256{return ErrMalformed}
 }
 if p.ChallengeID!="" && !decoded(p.ChallengeID,16){return ErrMalformed}
 return nil
}
func headerShape(h Header)error{
 if h.Type!=Type{return ErrUnsupported}
 if !signatures.Supported(h.Algorithm){return ErrUnsupported}
 if !key.MatchString(h.KeyID){return ErrMalformed}
 return nil
}
func timeShape(p Payload)error{
 // Upper bound year 9999 keeps every arithmetic expression in range.
 if p.Issued<=0 || p.Expires<=0 || p.Issued>253402300799 || p.Expires>253402300799 ||
  p.Expires<=p.Issued || p.Expires-p.Issued>86400{return ErrInvalidTime}
 if p.NotBefore!=nil && (*p.NotBefore<p.Issued || *p.NotBefore>=p.Expires){return ErrInvalidTime}
 return nil
}
// Issue signs a profile-v1 ticket using an explicitly chosen encrypted-file
// key loaded by the caller. Caller controls and must validate requested claims.
func Issue(p Payload,pair keys.Pair)(string,error){
 if e:=shape(p);e!=nil{return "",e}
 if e:=timeShape(p);e!=nil{return "",e}
 h:=Header{Type:Type,Algorithm:pair.Algorithm,KeyID:pair.ID}
 if e:=headerShape(h);e!=nil{return "",e}
 hb,e:=json.Marshal(h);if e!=nil{return "",ErrInternal}
 pb,e:=json.Marshal(p);if e!=nil{return "",ErrInternal}
 if len(hb)>512 || len(pb)>4096{return "",ErrMalformed}
 prefix:=base64.RawURLEncoding.EncodeToString(hb)+"."+base64.RawURLEncoding.EncodeToString(pb)
 sig,e:=signatures.Sign(pair.Algorithm,pair.Private,[]byte(prefix))
 if e!=nil{return "",fmt.Errorf("%w: signing unavailable",ErrInternal)}
 result:=prefix+"."+base64.RawURLEncoding.EncodeToString(sig)
 if len(result)>MaxToken{return "",ErrMalformed}
 return result,nil
}

// Verify accepts only cryptographically verified issuer-qualified evidence.
// The caller supplies the trusted audience/digest, never values copied from
// an unauthenticated client's claimed identity or upstream.
func Verify(compact,audience,invocationSHA256 string,registry *trust.Registry,now time.Time)(Evidence,error){
 var zero Evidence
 if len(compact)==0 || len(compact)>MaxToken{return zero,ErrMalformed}
 parts:=strings.Split(compact,".")
 if len(parts)!=3 || parts[0]==""||parts[1]==""||parts[2]==""{return zero,ErrMalformed}
 header,e:=base64.RawURLEncoding.DecodeString(parts[0])
 if e!=nil || len(header)==0 || len(header)>512 || base64.RawURLEncoding.EncodeToString(header)!=parts[0]{return zero,ErrMalformed}
 body,e:=base64.RawURLEncoding.DecodeString(parts[1])
 if e!=nil || len(body)==0 || len(body)>4096 || base64.RawURLEncoding.EncodeToString(body)!=parts[1]{return zero,ErrMalformed}
 signature,e:=base64.RawURLEncoding.DecodeString(parts[2])
 if e!=nil || len(signature)!=64 || base64.RawURLEncoding.EncodeToString(signature)!=parts[2]{return zero,ErrMalformed}
 var h Header
 if e=strictjson.Decode(header,&h,512);e!=nil{return zero,ErrMalformed}
 if e=headerShape(h);e!=nil{return zero,e}
 var p Payload
 if e=strictjson.Decode(body,&p,4096);e!=nil{return zero,ErrMalformed}
 if e=shape(p);e!=nil{return zero,e}
 if e=timeShape(p);e!=nil{return zero,e}
 if registry==nil{return zero,ErrUntrusted}
 pub,e:=registry.Lookup(p.Issuer,h.KeyID,h.Algorithm);if e!=nil{return zero,ErrUntrusted}
 if e=signatures.Verify(h.Algorithm,pub,[]byte(parts[0]+"."+parts[1]),signature);e!=nil{return zero,ErrInvalidSignature}
 if audience=="" || p.Audience!=audience || !decoded(invocationSHA256,32){return zero,ErrInvalidBinding}
 actual,_:=base64.RawURLEncoding.DecodeString(p.Invocation.SHA256)
 expected,_:=base64.RawURLEncoding.DecodeString(invocationSHA256)
 if !hmac.Equal(actual,expected){return zero,ErrInvalidBinding}
 nowSec:=now.UTC().Unix()
 if nowSec>=p.Expires || p.Issued>nowSec+30 || (p.NotBefore!=nil && *p.NotBefore>nowSec+30){return zero,ErrInvalidTime}
 claims:=make(map[string]string,len(p.Claims))
 for k,v:=range p.Claims{claims[k]=v}
 return Evidence{Issuer:p.Issuer,KeyID:h.KeyID,Audience:p.Audience,TicketID:p.ID,InvocationSHA256:p.Invocation.SHA256,
  Claims:claims,ChallengeID:p.ChallengeID,Expires:time.Unix(p.Expires,0).UTC()},nil
}
