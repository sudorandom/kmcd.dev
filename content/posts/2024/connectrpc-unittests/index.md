---
categories: ["article"]
tags: ["connectrpc", "grpc", "unittest"]
date: "2024-06-11T10:00:00Z"
description: "Learn how to test your ConnectRPC services."
cover: "cover.jpg"
title: "Unit Testing ConnectRPC Servers"
slug: "connectrpc-unittests"
type: "posts"
canonical_url: https://kmcd.dev/posts/connectrpc-unittests
draft: true
---

If you've decided to use [ConnectRPC](https://connectrpc.com/), you may be wondering how you can test your RPC services. Let's begin!

### Type assertion

```go
var _ v1connect.MyService = (*myService)(nil)
```

### 

```go
package service_test

import (
  "testing"

  "github.com/stretchr/testify/mock"

  "path/to/your/service"
  "path/to/your/mocks" // Mocks for dependencies
)

func TestMyService_DoSomething(t *testing.T) {
  // Create mock dependencies
  mockDB := new(mocks.MyDB)

  // Define mock behavior
  mockDB.On("GetData", "key").Return("data", nil)

  // Create service instance with mock
  myService := service.NewMyService(mockDB)

  // Call service method
  result, err := myService.DoSomething("key")

  // Assert on results
  if err != nil {
    t.Errorf("DoSomething returned error: %v", err)
  }

  if result != "data" {
    t.Errorf("DoSomething expected 'data', got %v", result)
  }

  // Verify mock expectations
  mockDB.AssertExpectations(t)
}

```