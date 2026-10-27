// Package strictjson decodes bounded JSON without accepting duplicate keys.
// Normal encoding/json happily accepts repeated object keys; trust and ticket
// data must not use ambiguous last-wins semantics.
package strictjson

import (
 "bytes"
 "encoding/json"
 "errors"
 "io"
 "unicode/utf8"
)
var ErrJSON=errors.New("invalid or ambiguous JSON")
func Decode(input []byte,target any,limit int)error{
 if len(input)==0 || len(input)>limit || !utf8.Valid(input){return ErrJSON}
 probe:=json.NewDecoder(bytes.NewReader(input))
 probe.UseNumber()
 if e:=walk(probe,0);e!=nil{return ErrJSON}
 if _,e:=probe.Token();e!=io.EOF{return ErrJSON}
 dec:=json.NewDecoder(bytes.NewReader(input))
 dec.DisallowUnknownFields()
 if e:=dec.Decode(target);e!=nil{return ErrJSON}
 var extra any
 if e:=dec.Decode(&extra);e!=io.EOF{return ErrJSON}
 return nil
}
func walk(dec *json.Decoder,depth int)error{
 if depth>64{return ErrJSON}
 token,e:=dec.Token();if e!=nil{return ErrJSON}
 delim,ok:=token.(json.Delim)
 if !ok{return nil}
 switch delim{
 case '{':
  seen:=map[string]bool{}
  for dec.More(){
   key,e:=dec.Token();if e!=nil{return ErrJSON}
   k,ok:=key.(string);if !ok||seen[k]{return ErrJSON}
   seen[k]=true
   if e:=walk(dec,depth+1);e!=nil{return e}
  }
  end,e:=dec.Token();if e!=nil||end!=json.Delim('}'){return ErrJSON}
 case '[':
  for dec.More(){if e:=walk(dec,depth+1);e!=nil{return e}}
  end,e:=dec.Token();if e!=nil||end!=json.Delim(']'){return ErrJSON}
 default:return ErrJSON
 }
 return nil
}
