# distobject-go

[![Build Status](https://github.com/didip/distobject-go/actions/workflows/test.yml/badge.svg)](https://github.com/didip/distobject-go/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/go-1.22-blue)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Distributed Object Library for Go (with Redis backend)

✅ Automatic dirty tracking  
✅ Save only changed fields  
✅ Load objects from Redis  
✅ Pub/Sub notifications on object updates  
✅ Concurrency safe  
✅ ULID based IDs

---

## ✨ Features

- Atomic partial updates via Redis `HSET`
- Auto-generate IDs using ULID
- Redis Pub/Sub notifications on field changes
- Reflective dirty tracking
- Concurrency safe for goroutines
- Easy integration into any Go project

---

## 🚀 Quick Start

```go
import (
    "github.com/didip/distobject-go/distobject"
)

type User struct {
    Name  string `redis:"name"`
    Email string `redis:"email"`
}

func main() {
    r := distobject.NewRedisClient("localhost:6379")
    user := &User{Name: "Alice", Email: "alice@example.com"}

    d := distobject.NewDistObject(r, "user", "user-updates")
    _ = d.Save(user)

    // Later load
    loaded := &User{}
    d2 := distobject.NewDistObject(r, "user", "user-updates")
    _ = d2.Load(d.ID(), loaded)

    fmt.Println(loaded.Name)  // Alice
}
```

## Running Tests Locally

First, start Redis:

```bash
docker compose up --build --abort-on-container-exit --exit-code-from test-runner
```

Or run tests manually:

```bash
make test
```

Run benchmarks:

```
make bench
```
