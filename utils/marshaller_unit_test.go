package utils

import (
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestMarshaller(t *testing.T) {
	req, err := http.NewRequest("GET", "https://www.google.com", strings.NewReader("z=post&both=y&prio=2&=nokey&orphan;empty=&"))
	if err != nil || req == nil {
		t.Fatalf("err making request\n%v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	req.Header.Set("Cookie", "name=xxxx; count=x")

	tr, err := TransformRequest(req)
	if err != nil {
		t.Fatalf("err transforming req\n%v", err)
	}

	r, err := UnTransformRequest(tr)
	if err != nil || r == nil {
		t.Fatalf("err untransforming request\n%v", err)
	}

	if req.Method != r.Method || req.Proto != r.Proto || req.ProtoMajor != r.ProtoMajor ||
		req.ProtoMinor != r.ProtoMinor || req.ContentLength != r.ContentLength || req.Close != r.Close ||
		req.Host != r.Host || req.RemoteAddr != r.RemoteAddr || req.RequestURI != r.RequestURI {
		t.Fatalf("expected:\n%v\n\nreceived:\n%v", req, r)
	}

	if !reflect.DeepEqual(req.TransferEncoding, r.TransferEncoding) {
		t.Fatalf("transfer encoding not equal\n%v\n%v", req.TransferEncoding, r.TransferEncoding)
	}
	if !reflect.DeepEqual(req.Form, r.Form) {
		t.Fatalf("form not equal\n%v\n%v", req.Form, r.Form)
	}
	if !reflect.DeepEqual(req.PostForm, r.PostForm) {
		t.Fatalf("postform not equal\n%v\n%v", req.PostForm, r.PostForm)
	}
	if !reflect.DeepEqual(req.Trailer, r.Trailer) {
		t.Fatalf("trailer not equal\n%v\n%v", req.Trailer, r.Trailer)
	}
	if !reflect.DeepEqual(req.Header, r.Header) {
		t.Fatalf("header not equal\n%v\n%v", req.Header, r.Header)
	}

	if r.URL != nil && req.URL != nil {
		if !reflect.DeepEqual(*r.URL, *req.URL) {
			t.Fatalf("expected:\n%v\n\nreceived:\n%v", *r.URL, *req.URL)
		}
	} else if (r.URL == nil && req.URL != nil) || (r.URL != nil && req.URL == nil) {
		t.Fatal("URL's are not equal")
	}
	if r.MultipartForm != nil && req.MultipartForm != nil {
		if !reflect.DeepEqual(*r.MultipartForm, *req.MultipartForm) {
			t.Fatalf("expected:\n%v\n\nreceived:\n%v", *r.MultipartForm, *req.MultipartForm)
		}
	} else if (r.MultipartForm == nil && req.MultipartForm != nil) || (r.MultipartForm != nil && req.MultipartForm == nil) {
		t.Fatalf("MultipartForm's are not equal\n%v\n%v", r.MultipartForm, req.MultipartForm)
	}
	if r.TLS != nil && req.TLS != nil {
		if !reflect.DeepEqual(*r.TLS, *req.TLS) {
			t.Fatalf("expected:\n%v\n\nreceived:\n%v", *r.TLS, *req.TLS)
		}
	} else if (r.TLS == nil && req.TLS != nil) || (r.TLS != nil && req.TLS == nil) {
		t.Fatal("TLS's are not equal")
	}
	if r.Response != nil && req.Response != nil {
		if !reflect.DeepEqual(*r.Response, *req.Response) {
			t.Fatalf("expected:\n%v\n\nreceived:\n%v", *r.Response, *req.Response)
		}
	} else if (r.Response == nil && req.Response != nil) || (r.Response != nil && req.Response == nil) {
		t.Fatal("Response's are not equal")
	}

	reqBody, err := req.GetBody()
	if err != nil {
		log.Fatalf("err getting req body\n%v", err)
	}

	rBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("err reading rBody\n%v", err)
	}
	reqBodyBytes, err := ioutil.ReadAll(reqBody)
	if err != nil {
		t.Fatalf("err reading reqBody\n%v", err)
	}

	if !reflect.DeepEqual(rBodyBytes, reqBodyBytes) {
		t.Fatal("Body bytes not equal")
	}
}
