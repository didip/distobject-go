package distobject

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type User struct {
	Name  string `redis:"name"`
	Email string `redis:"email"`
}

func setupRedis() *RedisClient {
	return NewRedisClient("redis:6379")
}

func TestNewRedisClient(t *testing.T) {
	r := setupRedis()
	assert.NotNil(t, r)
	pong, err := r.Client().Ping(ctx).Result()
	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
}

func TestNewDistObject(t *testing.T) {
	r := setupRedis()
	d := NewDistObject(r, "testuser", "testuser-updates")
	assert.NotNil(t, d)
	assert.Equal(t, "testuser", d.prefix)
	assert.Equal(t, "testuser-updates", d.channel)
}

func TestSaveLoadUser(t *testing.T) {
	r := setupRedis()

	user := &User{Name: "Alice", Email: "alice@example.com"}
	d := NewDistObject(r, "user", "user-updates")

	err := d.Save(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, d.ID())

	loaded := &User{}
	d2 := NewDistObject(r, "user", "user-updates")
	err = d2.Load(d.ID(), loaded)
	assert.NoError(t, err)

	assert.Equal(t, "Alice", loaded.Name)
	assert.Equal(t, "alice@example.com", loaded.Email)

	// Check internal state
	assert.NotEmpty(t, d.original["name"])
	assert.NotEmpty(t, d.original["email"])
	assert.NotEmpty(t, d.original["created_at"])
	assert.NotEmpty(t, d.original["updated_at"])
}

func TestMarkChanged(t *testing.T) {
	r := setupRedis()

	user := &User{Name: "Bob", Email: "bob@example.com"}
	d := NewDistObject(r, "user", "user-updates")

	// Manually mark name field dirty
	d.MarkChanged("name")

	// Save should capture name even without changing value
	err := d.Save(user)
	assert.NoError(t, err)
	assert.Contains(t, d.original, "name")
}

func TestGetID(t *testing.T) {
	r := setupRedis()

	user := &User{Name: "Charlie", Email: "charlie@example.com"}
	d := NewDistObject(r, "user", "user-updates")
	err := d.Save(user)
	assert.NoError(t, err)

	id := d.ID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "user:")
}

func TestConcurrentSaves(t *testing.T) {
	r := setupRedis()

	user := &User{Name: "Concurrent", Email: "initial@example.com"}
	d := NewDistObject(r, "user", "user-updates")
	err := d.Save(user)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	numThreads := 50

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := &User{}
			d2 := NewDistObject(r, "user", "user-updates")
			err := d2.Load(d.ID(), u)
			assert.NoError(t, err)

			u.Email = "user" + string(rune(i)) + "@example.com"
			d2.MarkChanged("email")
			err = d2.Save(u)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Final load
	finalUser := &User{}
	d3 := NewDistObject(r, "user", "user-updates")
	err = d3.Load(d.ID(), finalUser)
	assert.NoError(t, err)

	assert.NotEmpty(t, finalUser.Email)
}

func BenchmarkSave(b *testing.B) {
	r := setupRedis()

	for i := 0; i < b.N; i++ {
		user := &User{Name: "Bench", Email: "bench@example.com"}
		d := NewDistObject(r, "benchuser", "benchuser-updates")
		_ = d.Save(user)
	}
}

func BenchmarkLoad(b *testing.B) {
	r := setupRedis()

	user := &User{Name: "BenchLoad", Email: "benchload@example.com"}
	d := NewDistObject(r, "benchuser", "benchuser-updates")
	_ = d.Save(user)

	id := d.ID()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		loaded := &User{}
		d2 := NewDistObject(r, "benchuser", "benchuser-updates")
		_ = d2.Load(id, loaded)
	}
}
