package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

  "github.com/codegangsta/negroni"
  "github.com/gorilla/mux"
  "github.com/unrolled/render"
	"github.com/mitchellh/goamz/aws"
  "s3bench2/s3"
)

var rendering *render.Render

func int64toString(value int64) (string) {
	return strconv.FormatInt(value, 10)
}

type appError struct {
	err error
	status int
	json string
	template string
	binding interface{}
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if e := fn(w, r); e != nil {
		log.Print(e.err)
		if e.status != 0 {
			if e.json != "" {
				rendering.JSON(w, e.status, e.json)
			} else {
				rendering.HTML(w, e.status, e.template, e.binding)
			}
		}
  }
}

func RecoverHandler(next http.Handler) http.Handler {
  fn := func(w http.ResponseWriter, r *http.Request) {
    defer func() {
      if err := recover(); err != nil {
        log.Printf("panic: %+v", err)
        http.Error(w, http.StatusText(500), 500)
      }
    }()

    next.ServeHTTP(w, r)
  }
	return http.HandlerFunc(fn)
}

var s3Bucket *s3.Bucket

func main() {

	port := "8080"

  // See http://godoc.org/github.com/unrolled/render
  rendering = render.New(render.Options{Directory: "app/templates"})

  // See http://www.gorillatoolkit.org/pkg/mux
  router := mux.NewRouter()
  router.HandleFunc("/", Index).Methods("GET")
  router.Handle("/api/v1/requests/{key}", appHandler(Requests))
	router.PathPrefix("/app/").Handler(http.StripPrefix("/app/", http.FileServer(http.Dir("app"))))

	n := negroni.Classic()
	n.UseHandler(RecoverHandler(router))
	//http.ListenAndServeTLS(":" + port, "fe1b47ba5bcb246b.crt", "connectspeople.com.key", n)
	n.Run(":" + port)

	fmt.Printf("Listening on port " + port)
}

// Main page
func Index(w http.ResponseWriter, r *http.Request) {
  rendering.HTML(w, http.StatusOK, "index", nil)
}

type Response struct {
	Duration int64 `json:"duration"`
	StatusCode int `json:"statuscode"`
}

type Data struct {
	Timestamp string `json:"timestamp"`
	Listkeys Response `json:"listkeys"`
	PutCreate Response `json:"putcreate"`
	PutUpdate Response `json:"putupdate"`
	GetSame Response `json:"getsame"`
	GetRandom Response `json:"getrandom"`
	DeleteRandom Response `json:"deleterandom"`
	Endpoint string `json:"endpoint"`
	Bucket string `json:"bucket"`
}

// Execute requests
func Requests(w http.ResponseWriter, r *http.Request) *appError {
	decoder := json.NewDecoder(r.Body)
	var s map[string]string
	err := decoder.Decode(&s)
	if(err != nil) {
		fmt.Println(err)
	}

	s3Auth := aws.Auth{
    AccessKey: s["accesskey"],
    SecretKey: s["secretkey"],
  }

  s3SpecialRegion := aws.Region{
    Name: "Special",
    S3Endpoint: s["endpoint"],
  }

	s3BucketName := s["bucket"]

  s3Client := s3.New(s3Auth, s3SpecialRegion)
  s3Bucket = s3Client.Bucket(s3BucketName)

	vars := mux.Vars(r)
	timestamp := time.Now()
	uuid := int64toString(timestamp.UnixNano())
	hour, min, second := timestamp.Clock()
	data := Data{
		Timestamp: strconv.Itoa(hour) + ":" + strconv.Itoa(min) + ":" + strconv.Itoa(second),
		Endpoint: s["endpoint"],
		Bucket: s["bucket"],
	}

	if vars["key"] == "listkeys" {
		listkeysMessage := make(chan Response)

		go func() {
			start := time.Now()
			_, err := s3Bucket.List("", "", "", 0)
			statusCode := 200
			if(err != nil) {
				if(reflect.TypeOf(err) == reflect.TypeOf(&s3.Error{})) {
					statusCode = err.(*s3.Error).StatusCode
				} else if(reflect.TypeOf(err) == reflect.TypeOf(&url.Error{})) {
					statusCode = -1
				} else {
					fmt.Println(err)
					statusCode = -1
				}
			}
			duration := time.Now().Sub(start).Nanoseconds() / 1000000
			var response = Response{
				Duration: duration,
				StatusCode: statusCode,
			}
			listkeysMessage <- response
		}()

		listkeys := <- listkeysMessage

		data.Listkeys = listkeys
	}

	if vars["key"] == "putcreate" || vars["key"] == "putupdate" {
		putMessage := make(chan Response)

		go func() {
			start := time.Now()
			var err interface{}
			if vars["key"] == "putcreate" {
				err = s3Bucket.Put("/" + uuid + "1", []byte(uuid), "text/plain", "")
			} else {
				err = s3Bucket.Put("/object1", []byte(uuid), "text/plain", "")
			}
			statusCode := 201
			if(err != nil) {
				if(reflect.TypeOf(err) == reflect.TypeOf(&s3.Error{})) {
					statusCode = err.(*s3.Error).StatusCode
				} else if(reflect.TypeOf(err) == reflect.TypeOf(&url.Error{})) {
					statusCode = -1
				} else {
					fmt.Println(err)
					statusCode = -1
				}
			}
			duration := time.Now().Sub(start).Nanoseconds() / 1000000
			var response = Response{
				Duration: duration,
				StatusCode: statusCode,
			}
			putMessage <- response
		}()

		put := <- putMessage
		if vars["key"] == "putcreate" {
			data.PutCreate = put
		} else {
			data.PutUpdate = put
		}
	}

	if vars["key"] == "getsame" || vars["key"] == "getrandom" {
		getMessage := make(chan Response)

		go func() {
			var err interface{}
			if vars["key"] == "getrandom" {
				err = s3Bucket.Put("/" + uuid + "2", []byte(uuid), "text/plain", "")
			} else {
				err = s3Bucket.Put("/object2", []byte(uuid), "text/plain", "")
			}
			if err != nil {
				fmt.Println(err)
			}
			start := time.Now()
			if vars["key"] == "getrandom" {
				_, err = s3Bucket.Get("/" + uuid + "2")
			} else {
				_, err = s3Bucket.Get("/object2")
			}
			statusCode := 200
			if(err != nil) {
				if(reflect.TypeOf(err) == reflect.TypeOf(&s3.Error{})) {
					statusCode = err.(*s3.Error).StatusCode
				} else if(reflect.TypeOf(err) == reflect.TypeOf(&url.Error{})) {
					statusCode = -1
				} else {
					fmt.Println(err)
					statusCode = -1
				}
			}
			duration := time.Now().Sub(start).Nanoseconds() / 1000000
			var response = Response{
				Duration: duration,
				StatusCode: statusCode,
			}
			getMessage <- response
		}()

		get := <- getMessage

		if vars["key"] == "getrandom" {
			data.GetRandom = get
		} else {
			data.GetSame = get
		}
	}

	if vars["key"] == "deleterandom" {
		deleteMessage := make(chan Response)

		go func() {
			var err interface{}
			if vars["key"] == "deleterandom" {
				err = s3Bucket.Put("/" + uuid + "3", []byte(uuid), "text/plain", "")
			} else {
				err = s3Bucket.Put("/object3", []byte(uuid), "text/plain", "")
			}
			if err != nil {
				fmt.Println(err)
			}
			start := time.Now()
			if vars["key"] == "deleterandom" {
				err = s3Bucket.Del("/" + uuid + "3")
			} else {
				err = s3Bucket.Del("/object3")
			}
			statusCode := 204
			if(err != nil) {
				if(reflect.TypeOf(err) == reflect.TypeOf(&s3.Error{})) {
					statusCode = err.(*s3.Error).StatusCode
				} else if(reflect.TypeOf(err) == reflect.TypeOf(&url.Error{})) {
					statusCode = -1
				} else {
					fmt.Println(err)
					statusCode = -1
				}
			}
			duration := time.Now().Sub(start).Nanoseconds() / 1000000
			var response = Response{
				Duration: duration,
				StatusCode: statusCode,
			}
			deleteMessage <- response
		}()

		delete := <- deleteMessage

		if vars["key"] == "deleterandom" {
			data.DeleteRandom = delete
		}
	}

	rendering.JSON(w, http.StatusOK, data)

	return nil
}
