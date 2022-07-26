#### Overview:
A Protobuf Generator which generates from .proto files helper variables and functions which can be imported by microservices.

It can be installed with:
```
go install github.com/zyros90/protoc-gen-restapi@latest
```

if you have i.e. a `user.proto` file which have [google api proto annotations](https://github.com/googleapis/googleapis/blob/master/google/api/annotations.proto):
```proto
import "google/api/annotations.proto";
service UserSvc {
  rpc Create(CreateUserReq) returns (UserResp) {
    option (google.api.http) = {
      post : "/user/v1"
      body : "*"
    };
  };
```

it generates a `user.pb.api.go` file
```go
import (
	_ "github.com/labstack/echo/v4"
	restapi "github.com/zYros90/protoc-gen-restapi/utils"
)

const UserSvc_Create_Method = "POST"
const UserSvc_Create_Path = "/user/v1"

var UserSvcHTTP []*restapi.ApiAnnotations = []*restapi.ApiAnnotations{
	{
		Method: UserSvc_Create_Method,
		Path:   UserSvc_Create_Path,
	},
}
```

The generated file can be used in your service without rewriting the paths and methods redundantly.


#### Project State:
In development

#### Used Technologies:
* [protoc-gen-star](https://github.com/lyft/protoc-gen-star) efficient proto-based code generation
* [protoc-gen-go-http](https://github.com/go-kratos/kratos/tree/main/cmd/protoc-gen-go-http) for parsing google api annotations
* [text/template](https://pkg.go.dev/text/template) generating data-driven textual output
* [proto3](https://developers.google.com/protocol-buffers/docs/proto3)


#### TODOs
* Add generation of [echo](https://echo.labstack.com/) handler functions for registering routes