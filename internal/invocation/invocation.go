// Package invocation implements Euxenion's transport-independent, strict
// invocation binding profile. Values supplied as Principal and Upstream MUST
// be authenticated/configured by the caller, never copied from client JSON.
package invocation

import (
 "bytes"
 "crypto/sha256"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "math"
 "sort"
 "strconv"
 "strings"
 "unicode/utf8"
)

const Version = 1
const MaxArgumentsBytes = 1 << 20

var ErrInvalid = errors.New("invalid invocation descriptor")

// Descriptor is the exact operation to bind to a ticket. All identifiers
// must be supplied from trusted provenance; the library cannot authenticate them.
type Descriptor struct {
 Principal string
 Upstream string
 Method string
 Tool string
 Arguments json.RawMessage
}

// Canonical returns versioned deterministic bytes of a validated descriptor.
// Strict v1 profile: duplicate keys, invalid UTF-8, surrogate escapes and
// noncanonical/unsafe JSON numbers are rejected, not guessed.
func Canonical(d Descriptor) ([]byte, error) {
 if d.Principal == "" || d.Upstream == "" || d.Method == "" || d.Tool == "" ||
  !utf8.ValidString(d.Principal) || !utf8.ValidString(d.Upstream) ||
  !utf8.ValidString(d.Method) || !utf8.ValidString(d.Tool) {
  return nil, ErrInvalid
 }
 for _, s := range []string{d.Principal, d.Upstream, d.Method, d.Tool} {
  if len(s) > 256 { return nil, ErrInvalid }
 }
 if len(d.Arguments) == 0 || len(d.Arguments) > MaxArgumentsBytes || !utf8.Valid(d.Arguments) {
  return nil, ErrInvalid
 }
 arg, err := parseStrict(d.Arguments)
 if err != nil { return nil, fmt.Errorf("%w: %v", ErrInvalid, err) }
 root := map[string]any{"v":json.Number("1"), "principal":d.Principal,
  "upstream":d.Upstream,"method":d.Method,"tool":d.Tool,"arguments":arg}
 var out bytes.Buffer
 if err := encode(&out,root); err != nil { return nil, fmt.Errorf("%w: %v",ErrInvalid,err) }
 return out.Bytes(),nil
}

// Digest is base64url(SHA-256(Canonical(descriptor))) without padding.
func Digest(d Descriptor) (string,error) {
 b,e:=Canonical(d);if e!=nil{return "",e}
 h:=sha256.Sum256(b)
 return base64.RawURLEncoding.EncodeToString(h[:]),nil
}

func parseStrict(raw []byte)(any,error){
 if invalidSurrogates(raw){return nil,ErrInvalid}
 dec:=json.NewDecoder(bytes.NewReader(raw))
 dec.UseNumber()
 v,e:=readValue(dec,0);if e!=nil{return nil,e}
 if _,e=dec.Token();e!=io.EOF{return nil,ErrInvalid}
 return v,nil
}

// Escape sequences are checked before encoding/json can replace unpaired
// UTF-16 surrogates with U+FFFD. Only string literals are inspected.
func invalidSurrogates(raw []byte)bool{
 in,escaped:=false,false
 for i:=0;i<len(raw);i++{
  c:=raw[i]
  if !in { if c=='"' {in=true};continue }
  if !escaped { if c=='"' {in=false} else if c=='\\' {escaped=true};continue }
  escaped=false
  if c!='u' {continue}
  if i+4>=len(raw){return true}
  a,e:=strconv.ParseUint(string(raw[i+1:i+5]),16,16);if e!=nil{return true}
  i+=4
  if a>=0xdc00 && a<=0xdfff{return true}
  if a>=0xd800 && a<=0xdbff {
   if i+6>=len(raw) || raw[i+1]!='\\' || raw[i+2]!='u'{return true}
   b,e:=strconv.ParseUint(string(raw[i+3:i+7]),16,16)
   if e!=nil || b<0xdc00 || b>0xdfff{return true}
   i+=6
  }
 }
 return in || escaped
}
func readValue(dec *json.Decoder,depth int)(any,error){
 if depth>64 {return nil,ErrInvalid}
 t,e:=dec.Token();if e!=nil{return nil,ErrInvalid}
 switch v:=t.(type){
 case json.Delim:
  switch v{
  case '{':
   m:=make(map[string]any)
   for dec.More(){
    k,e:=dec.Token();if e!=nil{return nil,ErrInvalid}
    s,ok:=k.(string);if !ok{return nil,ErrInvalid}
    if _,exists:=m[s];exists{return nil,ErrInvalid}
    val,e:=readValue(dec,depth+1);if e!=nil{return nil,e}
    m[s]=val
   }
   end,e:=dec.Token();if e!=nil || end!=json.Delim('}') {return nil,ErrInvalid}
   return m,nil
  case '[':
   var a []any
   for dec.More(){val,e:=readValue(dec,depth+1);if e!=nil{return nil,e};a=append(a,val)}
   end,e:=dec.Token();if e!=nil || end!=json.Delim(']'){return nil,ErrInvalid}
   return a,nil
  default:return nil,ErrInvalid
  }
 case json.Number:
  if !isCanonicalNumber(string(v)){return nil,ErrInvalid}
  return v,nil
 case string,bool,nil:
  return v,nil
 default:return nil,ErrInvalid
 }
}

func isCanonicalNumber(s string)bool{
 f,e:=strconv.ParseFloat(s,64)
 if e!=nil || math.IsInf(f,0)||math.IsNaN(f)|| (f==0 && strings.HasPrefix(s,"-")){return false}
 // ECMAScript-compatible deterministic decimal profile. Reject alternate
 // representations instead of silently rounding numerically distinct inputs.
 canonical:=strconv.FormatFloat(f,'f',-1,64)
 if math.Abs(f)<1e-6 && f!=0 || math.Abs(f)>=1e21 {
  canonical=strconv.FormatFloat(f,'e',-1,64)
  parts:=strings.Split(canonical,"e")
  exp,_:=strconv.Atoi(parts[1])
  canonical=parts[0]+"e"+strconv.FormatInt(int64(exp),10)
  if exp>=0 {canonical=parts[0]+"e+"+strconv.Itoa(exp)}
 }
 return s==canonical
}

func encode(out *bytes.Buffer,v any)error{
 switch x:=v.(type) {
 case nil:out.WriteString("null")
 case bool:if x{out.WriteString("true")}else{out.WriteString("false")}
 case string:
  b,_:=json.Marshal(x)
  out.Write(b)
 case json.Number:out.WriteString(string(x))
 case []any:
  out.WriteByte('[')
  for i,e:=range x{if i!=0{out.WriteByte(',')};if err:=encode(out,e);err!=nil{return err}}
  out.WriteByte(']')
 case map[string]any:
  keys:=make([]string,0,len(x));for k:=range x{keys=append(keys,k)}
  sort.Strings(keys);out.WriteByte('{')
  for i,k:=range keys{
   if i!=0{out.WriteByte(',')}
   if err:=encode(out,k);err!=nil{return err}
   out.WriteByte(':')
   if err:=encode(out,x[k]);err!=nil{return err}
  }
  out.WriteByte('}')
 default:return ErrInvalid
 }
 return nil
}
