FROM golang:1.3
RUN git clone github.com/djannot/s3bench2.git
WORKDIR /go/s3bench2
RUN go get github.com/tools/godep
RUN go get "github.com/codegangsta/negroni"
RUN go get "github.com/gorilla/mux"
RUN go get "github.com/mitchellh/goamz/aws"
RUN go get "github.com/unrolled/render"
RUN godep save
RUN go build
