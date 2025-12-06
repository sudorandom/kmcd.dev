---
categories: ["article"]
tags: ["grpc", "connectrpc", "openapi", "protobuf", "protoc", "rpc", "go", "golang"]
date: "2024-09-25T10:00:00Z"
description: "gRPC can be pretty, too."
cover: "cover.jpg"
images: ["/posts/self-documenting-connect-services/cover.jpg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "Self-Documenting Connect Services"
slug: "self-documenting-connect-services"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/self-documenting-connect-services/
mastodonID: "113197627641199319"
---

As some of you may know, I've created a plugin for protoc called [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi). This plugin converts protobuf files into [OpenAPI specifications](https://swagger.io/specification/) for [the Connect protocol](https://connectrpc.com/docs/protocol/). This protocol is very similar to gRPC but for unary RPCs it follows many more traditions that you'd expect from an HTTP-based API, like using HTTP status codes appropriately, using the normal `Content-Encoding` header to specify compression and avoiding putting extra framing inside of the body. Because of this, we can document it more readily with other specifications like OpenAPI.

For more on the plugin itself, refer to [my older post](/posts/protoc-gen-connect-openapi/) that introduces protoc-gen-connect-openapi or [the github repo](https://github.com/sudorandom/protoc-gen-connect-openapi) which has some more updates.

This post is about a new way to use this functionality, from [a Go library](https://pkg.go.dev/github.com/sudorandom/protoc-gen-connect-openapi/converter). This Go API provides a simpler interface to generate OpenAPI from protobuf descriptors. You see, protoc plugins accept a `*pluginpb.CodeGeneratorRequest` and return a `*pluginpb.CodeGeneratorResponse`. The request type, in particular, is hard to use from Go. You have to encode every option you want to use into a single string. This is fine for a CLI but it's not very friendly for a Go library. But now that I made this library it is much easier to generate OpenAPI specs anywhere you run Go. Let's look at some examples.

## Generating OpenAPI
First, let's see how we can use this plugin to generate OpenAPI YAML from your protobuf definitions:

```go
openapiBody, _ := converter.GenerateSingle(
    converter.WithGlobal(),
    converter.WithBaseOpenAPI([]byte(`
openapi: 3.1.0
info:
  title: OpenAPI Documentation of gRPC Services
  description: This is documentation that was generated from [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi).
  version: 0.1.2
`)))
fmt.Println(string(openapiBody))
```
With a few short lines, you now have OpenAPI YAML representation of your ConnectRPC services.

```yaml
openapi: 3.1.0
info:
  title: OpenAPI Documentation of gRPC Services
  description: This is documentation that was generated from [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi).
paths:
  /connectrpc.eliza.v1.ElizaService/Say:
    post:
      tags:
        - connectrpc.eliza.v1.ElizaService
      summary: Say
      operationId: connectrpc.eliza.v1.ElizaService.Say
      parameters:
        - name: Connect-Protocol-Version
          in: header
          required: true
          schema:
            $ref: '#/components/schemas/connect-protocol-version'
        - name: Connect-Timeout-Ms
          in: header
          schema:
            $ref: '#/components/schemas/connect-timeout-header'
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/connectrpc.eliza.v1.SayRequest'
        required: true
      responses:
        default:
          description: Error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/connect.error'
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/connectrpc.eliza.v1.SayResponse'
  /connectrpc.eliza.v1.ElizaService/Converse: {}
  /connectrpc.eliza.v1.ElizaService/Introduce: {}
components:
  schemas:
    connectrpc.eliza.v1.SayRequest:
      type: object
      properties:
        sentence:
          type: string
          title: sentence
      title: SayRequest
      additionalProperties: false
    connectrpc.eliza.v1.SayResponse:
      type: object
      properties:
        sentence:
          type: string
          title: sentence
      title: SayResponse
      additionalProperties: false
    connect-protocol-version:
      type: number
      title: Connect-Protocol-Version
      enum:
        - 1
      description: Define the version of the Connect protocol
      const: 1
    connect-timeout-header:
      type: number
      title: Connect-Timeout-Ms
      description: Define the timeout, in ms
    connect.error:
      type: object
      properties:
        code:
          type: string
          examples:
            - CodeNotFound
          enum:
            - CodeCanceled
            - CodeUnknown
            - CodeInvalidArgument
            - CodeDeadlineExceeded
            - CodeNotFound
            - CodeAlreadyExists
            - CodePermissionDenied
            - CodeResourceExhausted
            - CodeFailedPrecondition
            - CodeAborted
            - CodeOutOfRange
            - CodeInternal
            - CodeUnavailable
            - CodeDataLoss
            - CodeUnauthenticated
          description: The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].
        message:
          type: string
          description: A developer-facing error message, which should be in English. Any user-facing error message should be localized and sent in the [google.rpc.Status.details][google.rpc.Status.details] field, or localized by the client.
        detail:
          $ref: '#/components/schemas/google.protobuf.Any'
      title: Connect Error
      additionalProperties: true
      description: 'Error type returned by Connect: https://connectrpc.com/docs/go/errors/#http-representation'
    google.protobuf.Any:
      type: object
      properties:
        type:
          type: string
        value:
          type: string
          format: binary
        debug:
          type: object
          additionalProperties: true
      additionalProperties: true
      description: Contains an arbitrary serialized message along with a @type that describes the type of the serialized message.
security: []
tags:
  - name: connectrpc.eliza.v1.ElizaService
```

This file is truncated to only show a single endpoint and related types. To see the full file, [click here](/posts/self-documenting-connect-services/openapi.yaml). The `converter.WithGlobal()` option uses the global protobuf registry as the source. If you want to use specific (or protobuf file descriptors not in that registry), you can pass in any `protoregistry.GlobalFiles` value to the `converter.WithFiles()` option. With `converter.WithBaseOpenAPI()`, you can specify a base OpenAPI spec that will be used as the basis for the generated one. Here you can add a description, version, security schemes, link to other documentation, etc.

But what's the practical use of this YAML?

## Show the world what you can do
With a few additional lines of code, we can leverage one of the numerous OpenAPI documentation visualization tools to transform this YAML into a visually appealing and interactive web page. Here's an example using Elements from Spotlight:

```go
var tmplElements = template.Must(template.New("name").Parse(`<!doctype html>
<html lang="en">
	<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
	<title>OpenAPI Documentation</title>
	<script src="https://unpkg.com/@stoplight/elements@8.3.4/web-components.min.js"></script>
	<link rel="stylesheet" href="https://unpkg.com/@stoplight/elements@8.3.4/styles.min.css">
	</head>
	<body>

	<elements-api
		id="docs"
		router="hash"
		layout="sidebar"
	/>
	<script>
	(async () => {
		const docs = document.getElementById('docs');
		docs.apiDescriptionDocument = atob("{{ .DocumentBase64 }}");
	})();
	</script>

	</body>
</html>`))

func main() {
	mux := http.NewServeMux()
	mux.Handle(elizav1connect.NewElizaServiceHandler(&elizav1connect.UnimplementedElizaServiceHandler{}))
	openapiBody, err := converter.GenerateSingle(
		converter.WithGlobal(),
		converter.WithContentTypes(
			"json",
			"proto",
		),
		converter.WithStreaming(true),
        converter.WithAllowGET(true),
		converter.WithBaseOpenAPI([]byte(`
openapi: 3.1.0
info:
  title: OpenAPI Documentation of gRPC Services
  description: This is documentation that was generated from [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi).
`)))
	if err != nil {
		log.Fatalf("err: %s", err)
	}
	generationTime := time.Now()

	mux.Handle("GET /openapi.html", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := tmplElements.Execute(w, struct{ DocumentBase64 string }{
			DocumentBase64: base64.StdEncoding.EncodeToString(openapiBody),
		}); err != nil {
			slog.Error("rendering_template", "error", err)
		}
	}))
	mux.Handle("GET /openapi.yaml", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "openapi.yaml", generationTime, bytes.NewReader(openapiBody))
	}))

	addr := "127.0.0.1:6660"
	log.Printf("Starting connectrpc on http://%s", addr)
	log.Printf("OpenAPI Doc Page http://%s/openapi.html", addr)
	log.Printf("OpenAPI Spec http://%s/openapi.yaml", addr)
	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("error: %s", err)
	}
}
```

And here's what it looks like whenever you hit `http://127.0.0.1.:6660/openapi.html` in a web browser:

{{< image src="openapi-sshot.png" width="800px" class="center" >}}

To see the demo for yourself, [click here!](/posts/self-documenting-connect-services/openapi.html)

In this example, we're using the [@stoplight/elements](https://www.npmjs.com/package/@stoplight/elements) library to render the OpenAPI documentation. The `tmplElements` template embeds the generated OpenAPI YAML (Base64 encoded) into an HTML page, providing a user-friendly interface to explore your API's endpoints, request/response structures, and more. Note that we're using base64 so that none of the YAML characters accidentally escape the javascript string and mess everything up for us. You can also have the script load the OpenAPI spec from a URL, which is what [many of the examples show](https://github.com/stoplightio/elements?tab=readme-ov-file#web-component).

You may also notice some additional options in this example, like `converter.WithContentTypes()`, `converter.WithStreaming(true)` and `converter.WithAllowGET(true)`. These give you more control over content types, whether you want OpenAPI for streaming calls (which may be complicated to support for OpenAPI) and whether you want to generate documentation for GET requests, that Connect supports if you can set the `idempotency_level` option to `NO_SIDE_EFFECTS`. For more information on GET requests and a comprehensive list of available options, refer to the [the Connect documentation](https://connectrpc.com/docs/go/get-requests-and-caching/) and the [protoc-gen-connect-openapi Go documentation](https://pkg.go.dev/github.com/sudorandom/protoc-gen-connect-openapi/converter), respectively.

## Benefits of Self-Documenting Services
Clear and interactive documentation makes it easier for developers to understand and integrate with your APIs. This reduces the friction between teams and gives you something to point to in case there are questions about the API. Not everyone is fluent in reading protobuf, but HTTP documentation is much friendlier.

By generating documentation from protobufs, it is much easier for documentation to stay in sync with your codebase. Note that adding comments to your protobuf types, fields, services and methods also get carried over to this OpenAPI specification (and the generated protobuf/gRPC/gRPC-Web/Connect source code) so protobuf files can act as a single place to document everything about that service.

## Conclusion
By combining the power of protoc-gen-connect-openapi with OpenAPI visualization tools, you can effortlessly generate self-documenting Connect services. This approach streamlines development, fosters collaboration, and empowers developers to consume your APIs effectively.
