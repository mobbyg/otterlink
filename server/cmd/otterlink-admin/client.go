package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token string
	username string
	password string
	http *http.Client
}

func NewClient(c Config) *Client {
	return &Client{baseURL: strings.TrimRight(c.Server, "/"), token:c.Token, username:c.Username, password:c.Password, http:&http.Client{Timeout:20*time.Second}}
}

func (c *Client) EnsureAuth(interactive bool) error {
	if c.token != "" { return nil }
	if c.username == "" {
		if !interactive { return fmt.Errorf("no session token; set OTTERLINK_TOKEN or use --token") }
		fmt.Print("Username: ")
		in:=bufio.NewReader(os.Stdin); v,_:=in.ReadString('\n'); c.username=strings.TrimSpace(v)
	}
	if c.password == "" {
		if !interactive { return fmt.Errorf("no password; set OTTERLINK_PASSWORD for non-interactive login") }
		var err error; c.password,err=readSecret("Password: "); if err!=nil{return err}
	}
	var resp map[string]any
	if err:=c.request("POST","/api/auth/login",map[string]any{"username":c.username,"password":c.password},&resp);err!=nil{return err}
	user,_:=resp["user"].(map[string]any); role,_:=user["role"].(string); name,_:=user["username"].(string)
	if role!="admin" { return fmt.Errorf("account %q is not an administrator",name) }
	c.token,_=resp["token"].(string)
	if c.token=="" { return fmt.Errorf("login succeeded but server returned no token") }
	c.password=""
	return nil
}

func readSecret(prompt string)(string,error){
	fmt.Print(prompt)
	_ = exec.Command("stty","-echo").Run()
	defer exec.Command("stty","echo").Run()
	v,err:=bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	return strings.TrimSpace(v),err
}

func (c *Client) request(method,path string,body any,dst any) error {
	var r io.Reader
	if body!=nil { b,err:=json.Marshal(body);if err!=nil{return err};r=bytes.NewReader(b) }
	req,err:=http.NewRequest(method,c.baseURL+path,r);if err!=nil{return err}
	if body!=nil{req.Header.Set("Content-Type","application/json")}
	if c.token!=""{req.Header.Set("Authorization","Bearer "+c.token)}
	res,err:=c.http.Do(req);if err!=nil{return err};defer res.Body.Close()
	data,_:=io.ReadAll(res.Body)
	if res.StatusCode<200||res.StatusCode>=300{msg:=strings.TrimSpace(string(data));if msg==""{msg=res.Status};return fmt.Errorf("%s: %s",res.Status,msg)}
	if dst==nil||len(data)==0{return nil}
	if err:=json.Unmarshal(data,dst);err!=nil{return fmt.Errorf("decode server response: %w",err)}
	return nil
}
func(c *Client)get(path string,dst any)error{return c.request("GET",path,nil,dst)}
func(c *Client)post(path string,body,dst any)error{return c.request("POST",path,body,dst)}
func(c *Client)patch(path string,body,dst any)error{return c.request("PATCH",path,body,dst)}
func(c *Client)delete(path string,body,dst any)error{return c.request("DELETE",path,body,dst)}
