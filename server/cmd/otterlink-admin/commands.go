package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func userCommand(c *Client,a []string)(Records,error){
	if len(a)==0{return nil,fmt.Errorf("usage: users <list|get|update|password|revoke|delete>")}
	switch a[0]{
	case "list":
		var x struct{Users []map[string]any \`json:"users"\`};if err:=c.get("/api/admin/users",&x);err!=nil{return nil,err};return mapsToRecords(x.Users),nil
	case "get":
		if len(a)!=2{return nil,fmt.Errorf("usage: users get USER")};var x map[string]any;if err:=c.get("/api/admin/users/"+url.PathEscape(a[1]),&x);err!=nil{return nil,err};return Records{x},nil
	case "update":
		if len(a)<2{return nil,fmt.Errorf("usage: users update USER [flags]")};m,err:=flags(a[2:]);if err!=nil{return nil,err}
		var current map[string]any
		if err:=c.get("/api/admin/users/"+url.PathEscape(a[1]),&current);err!=nil{return nil,err}
		body:=map[string]any{"display_name":current["display_name"],"email":current["email"],"status":current["status"],"role":current["role"]}
		if v,ok:=m["display-name"];ok{body["display_name"]=v};if v,ok:=m["email"];ok{body["email"]=v};if v,ok:=m["status"];ok{body["status"]=v};if v,ok:=m["role"];ok{body["role"]=v}
		var x map[string]any;if err:=c.patch("/api/admin/users/"+url.PathEscape(a[1]),body,&x);err!=nil{return nil,err};return Records{x},nil
	case "password":
		if len(a)!=2{return nil,fmt.Errorf("usage: users password USER")};p,err:=readSecret("New password: ");if err!=nil{return nil,err};if len(p)<12{return nil,fmt.Errorf("password must be at least 12 characters")};var x map[string]any;if err:=c.post("/api/admin/users/"+url.PathEscape(a[1])+"/password",map[string]any{"password":p},&x);err!=nil{return nil,err};return Records{x},nil
	case "revoke":
		if len(a)!=2{return nil,fmt.Errorf("usage: users revoke USER")};if err:=c.post("/api/admin/users/"+url.PathEscape(a[1])+"/sessions/revoke",nil,nil);err!=nil{return nil,err};return Records{{"ok":true}},nil
	case "delete":
		if len(a)!=2{return nil,fmt.Errorf("usage: users delete USER")};p,err:=readSecret("Admin password: ");if err!=nil{return nil,err};if err:=c.delete("/api/admin/users/"+url.PathEscape(a[1]),map[string]any{"password":p},nil);err!=nil{return nil,err};return Records{{"ok":true}},nil
	default:return nil,fmt.Errorf("unknown users command %q",a[0])
	}
}
func chatCommand(c *Client,a []string)(Records,error){
	if len(a)==0{return nil,fmt.Errorf("usage: chat <list|create|delete|role>")}
	switch a[0]{
	case "list":
		var x struct{Channels []map[string]any \`json:"channels"\`};if err:=c.get("/api/admin/chat/channels",&x);err!=nil{return nil,err};return mapsToRecords(x.Channels),nil
	case "create":
		if len(a)<2{return nil,fmt.Errorf("usage: chat create NAME [--allow-ops]")};m,err:=flags(a[2:]);if err!=nil{return nil,err};var x map[string]any;if err:=c.post("/api/admin/chat/channels",map[string]any{"name":a[1],"allow_ops_to_create_ops":m["allow-ops"]=="true"},&x);err!=nil{return nil,err};return Records{x},nil
	case "delete":
		if len(a)!=2{return nil,fmt.Errorf("usage: chat delete ID")};if _,err:=strconv.ParseInt(a[1],10,64);err!=nil{return nil,fmt.Errorf("invalid channel id")};if err:=c.delete("/api/admin/chat/channels/"+url.PathEscape(a[1]),nil,nil);err!=nil{return nil,err};return Records{{"ok":true}},nil
	case "role":
		if len(a)!=4{return nil,fmt.Errorf("usage: chat role CHANNEL_ID USER mod|op|user")};if err:=c.post("/api/admin/chat/channels/"+url.PathEscape(a[1])+"/users/"+url.PathEscape(a[2])+"/role",map[string]any{"role":a[3]},nil);err!=nil{return nil,err};return Records{{"ok":true}},nil
	default:return nil,fmt.Errorf("unknown chat command %q",a[0])
	}
}
func eventCommand(c *Client,a []string)(Records,error){
	if len(a)==0{return nil,fmt.Errorf("usage: events <list|create|update|delete>")}
	switch a[0]{
	case "list":
		m,err:=flags(a[1:]);if err!=nil{return nil,err};path:="/api/admin/events";q:=[]string{};if m["year"]!=""{q=append(q,"year="+url.QueryEscape(m["year"]))};if m["month"]!=""{q=append(q,"month="+url.QueryEscape(m["month"]))};if len(q)>0{path+="?"+strings.Join(q,"&")};var x struct{Events []map[string]any \`json:"events"\`};if err:=c.get(path,&x);err!=nil{return nil,err};return mapsToRecords(x.Events),nil
	case "create","update":
		if a[0]=="update"&&len(a)<2{return nil,fmt.Errorf("usage: events update ID [flags]")};start:=1;if a[0]=="update"{start=2};m,err:=flags(a[start:]);if err!=nil{return nil,err};body:=map[string]any{"title":m["title"],"description":m["description"],"start_at":m["start-at"],"end_at":m["end-at"],"all_day":m["all-day"]=="true"};if body["title"]==nil||body["start_at"]==nil||body["end_at"]==nil{return nil,fmt.Errorf("--title, --start-at and --end-at are required")};if a[0]=="create"{body["event_type"]="server";body["target_type"]="none";body["target_id"]=0};var x map[string]any;var e error;if a[0]=="create"{e=c.post("/api/admin/events",body,&x)}else{e=c.patch("/api/admin/events/"+url.PathEscape(a[1]),body,&x)};if e!=nil{return nil,e};return Records{x},nil
	case "delete":
		if len(a)!=2{return nil,fmt.Errorf("usage: events delete ID")};if err:=c.delete("/api/admin/events/"+url.PathEscape(a[1]),nil,nil);err!=nil{return nil,err};return Records{{"ok":true}},nil
	default:return nil,fmt.Errorf("unknown events command %q",a[0])
	}
}
func auditCommand(c *Client,a []string)(Records,error){if len(a)!=1||a[0]!="list"{return nil,fmt.Errorf("usage: audit list")};var x struct{Events []map[string]any \`json:"events"\`};if err:=c.get("/api/admin/audit",&x);err!=nil{return nil,err};return mapsToRecords(x.Events),nil}
func presenceCommand(c *Client,a []string)(Records,error){if len(a)!=1||a[0]!="list"{return nil,fmt.Errorf("usage: presence list")};var x struct{Users []map[string]any \`json:"users"\`};if err:=c.get("/api/presence",&x);err!=nil{return nil,err};return mapsToRecords(x.Users),nil}
func mapsToRecords(v []map[string]any)Records{r:=make(Records,len(v));for i,m:=range v{r[i]=Record(m)};return r}
func flags(a []string)(map[string]string,error){out:=map[string]string{};for i:=0;i<len(a);i++{if !strings.HasPrefix(a[i],"--"){return nil,fmt.Errorf("expected flag, got %q",a[i])};p:=strings.TrimPrefix(a[i],"--");if i+1<len(a)&&!strings.HasPrefix(a[i+1],"--"){out[p]=a[i+1];i++}else{out[p]="true"}};return out,nil}
