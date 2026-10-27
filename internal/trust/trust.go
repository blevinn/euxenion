// Package trust loads a strictly configured set of issuer-qualified public
// keys. A verified signature is evidence, never permission to dispatch.
package trust

import (
 "crypto"
 "errors"
 "os"
 "path/filepath"
 "regexp"

 "github.com/blevinn/euxenion/internal/keys"
 "github.com/blevinn/euxenion/internal/signatures"
 "github.com/blevinn/euxenion/internal/strictjson"
)

var ErrTrust = errors.New("invalid or untrusted issuer configuration")
var issuerID=regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

type KeyConfig struct {
 ID string `json:"kid"`
 Algorithm signatures.Algorithm `json:"alg"`
 PublicFile string `json:"public_key"`
}
type IssuerConfig struct {
 ID string `json:"id"`
 Keys []KeyConfig `json:"keys"`
}
type Config struct {
 Version int `json:"v"`
 Issuers []IssuerConfig `json:"issuers"`
}
type Key struct{
 Algorithm signatures.Algorithm
 Public crypto.PublicKey
}
// Registry is immutable after loading. Each key is namespaced by issuer.
type Registry struct{ issuers map[string]map[string]Key }

func LoadFile(path string)(*Registry,error){
 info,e:=os.Lstat(path)
 if e!=nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022!=0{return nil,ErrTrust}
 data,e:=os.ReadFile(path);if e!=nil{return nil,ErrTrust}
 return Load(data,filepath.Dir(path))
}
func Load(data []byte,root string)(*Registry,error){
 var cfg Config
 if e:=strictjson.Decode(data,&cfg,1<<20);e!=nil || cfg.Version!=1 || len(cfg.Issuers)==0 || len(cfg.Issuers)>64{return nil,ErrTrust}
 out:=&Registry{issuers:make(map[string]map[string]Key)}
 for _,issuer:=range cfg.Issuers{
  if !issuerID.MatchString(issuer.ID) || out.issuers[issuer.ID]!=nil || len(issuer.Keys)==0 || len(issuer.Keys)>64{return nil,ErrTrust}
  entries:=make(map[string]Key)
  for _,item:=range issuer.Keys{
   if !signatures.Supported(item.Algorithm) || !filepath.IsLocal(item.PublicFile) || entries[item.ID].Public!=nil{return nil,ErrTrust}
   path:=filepath.Join(root,item.PublicFile)
   info,e:=os.Lstat(path)
   if e!=nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022!=0{return nil,ErrTrust}
   pem,e:=os.ReadFile(path);if e!=nil{return nil,ErrTrust}
   pub,alg,id,e:=keys.ParsePublic(pem)
   if e!=nil || alg!=item.Algorithm || id!=item.ID{return nil,ErrTrust}
   entries[id]=Key{Algorithm:alg,Public:pub}
  }
  out.issuers[issuer.ID]=entries
 }
 return out,nil
}
func (r *Registry) Lookup(issuer,kid string,alg signatures.Algorithm)(crypto.PublicKey,error){
 if r==nil{return nil,ErrTrust}
 keysForIssuer,ok:=r.issuers[issuer];if !ok{return nil,ErrTrust}
 entry,ok:=keysForIssuer[kid]
 if !ok || entry.Algorithm!=alg || entry.Public==nil{return nil,ErrTrust}
 return entry.Public,nil
}
