package distobject

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var ctx = context.Background()

type DistObject struct {
	redis    *RedisClient
	prefix   string
	channel  string
	id       string
	original map[string]string
	changed  map[string]bool
}

func NewDistObject(r *RedisClient, prefix string, channel string) *DistObject {
	return &DistObject{
		redis:    r,
		prefix:   prefix,
		channel:  channel,
		original: make(map[string]string),
		changed:  make(map[string]bool),
	}
}

// Utility: generate ULID
func generateULID() string {
	entropy := ulid.Monotonic(nil, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return id.String()
}

// Save the struct into Redis
func (d *DistObject) Save(obj interface{}) error {
	if d.id == "" {
		d.id = fmt.Sprintf("%s:%s", d.prefix, generateULID())
	}

	v := reflect.ValueOf(obj).Elem()
	t := v.Type()

	fields := make(map[string]interface{})

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("redis")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		value := fmt.Sprintf("%v", v.Field(i).Interface())

		// Track only changed fields
		if oldVal, exists := d.original[tag]; !exists || oldVal != value || d.changed[tag] {
			fields[tag] = value
		}
	}

	// Add timestamps
	now := fmt.Sprintf("%d", time.Now().Unix())
	fields["updated_at"] = now
	if _, ok := d.original["created_at"]; !ok {
		fields["created_at"] = now
	}

	_, err := d.redis.Client().HSet(ctx, d.id, fields).Result()
	if err != nil {
		return err
	}

	// Publish change
	notif := map[string]interface{}{
		"id":      d.id,
		"changes": fields,
	}
	notifBytes, _ := json.Marshal(notif)
	_ = d.redis.Client().Publish(ctx, d.channel, notifBytes).Err()

	// Update original
	for k, v := range fields {
		d.original[k] = fmt.Sprintf("%v", v)
	}
	d.changed = make(map[string]bool) // clear dirty tracking

	return nil
}

// Load object from Redis
func (d *DistObject) Load(id string, obj interface{}) error {
	data, err := d.redis.Client().HGetAll(ctx, id).Result()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("object not found: %s", id)
	}

	v := reflect.ValueOf(obj).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("redis")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		if val, exists := data[tag]; exists {
			v.Field(i).Set(reflect.ValueOf(val))
		}
	}

	d.id = id
	d.original = data
	d.changed = make(map[string]bool)

	return nil
}

// Mark a field dirty manually
func (d *DistObject) MarkChanged(field string) {
	d.changed[field] = true
}

// Get ID
func (d *DistObject) ID() string {
	return d.id
}
