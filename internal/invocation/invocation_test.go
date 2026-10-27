package invocation

import (
 "encoding/json"
 "errors"
 "strings"
 "testing"
)

func example() Descriptor{
 return Descriptor{Principal:"local:operator",Upstream:"upstream:one",Method:"tools/call",Tool:"write",Arguments:json.RawMessage(`{"b":[true,5],"a":"Ω"}`)}
}
func TestDigestBinding(t *testing.T){
 d:=example()
 first,e:=Digest(d);if e!=nil{t.Fatal(e)}
 if len(first)!=43{t.Fatalf("unexpected digest %q",first)}
 d.Arguments=json.RawMessage(`{"a":"Ω","b":[true,5]}`)
 second,e:=Digest(d);if e!=nil || first!=second{t.Fatalf("unstable digest %q %q %v",first,second,e)}
 for _,mut:=range []func(*Descriptor){func(x *Descriptor){x.Principal="other"},func(x *Descriptor){x.Upstream="other"},func(x *Descriptor){x.Method="resources/read"},func(x *Descriptor){x.Tool="delete"},func(x *Descriptor){x.Arguments=json.RawMessage(`{"a":"Ω","b":[true,6]}`)}}{
  x:=example();mut(&x);v,e:=Digest(x);if e!=nil||v==first{t.Fatalf("did not rebind: %q %v",v,e)}
 }
}
func TestRejectAmbiguity(t *testing.T){
 cases:=[]string{
  `{"a":1,"a":2}`,`{"x":{"a":1,"a":2}}`,`{"x":1.0}`,
  `{"x":1e3}`,`{"x":-0}`,`{"x":9007199254740993}`,
  `{"x":1e9999}`,`{"x":"\uD800"}`,`{"x":"\uDC00"}`,
  `{"x":1} trailing`,`{"x":}`,`[`,
 }
 for _,a:=range cases {t.Run(a,func(t *testing.T){d:=example();d.Arguments=[]byte(a);_,e:=Digest(d);if !errors.Is(e,ErrInvalid){t.Fatalf("expected rejection: %v",e)}})}
}
func TestUnicodeAndNoMeta(t *testing.T){
 d:=example();d.Arguments=[]byte(`{"text":"\u03A9","_meta":{"ignored":"included when an actual tool argument"}}`)
 a,e:=Digest(d);if e!=nil{t.Fatal(e)}
 d.Arguments=[]byte(`{"_meta":{"ignored":"included when an actual tool argument"},"text":"Ω"}`)
 b,e:=Digest(d);if e!=nil||a!=b{t.Fatal("unicode escape mismatch",e)}
 d.Arguments=[]byte(`{"text":"Ω"}`)
 c,e:=Digest(d);if e!=nil||a==c{t.Fatal("tool argument was dropped",e)}
 if _,e=Canonical(Descriptor{Arguments:json.RawMessage("{}")});!errors.Is(e,ErrInvalid){t.Fatal(e)}
 if _,e=Canonical(Descriptor{Principal:strings.Repeat("x",257),Upstream:"u",Method:"m",Tool:"t",Arguments:json.RawMessage("{}")});e==nil{t.Fatal("length accepted")}
}
