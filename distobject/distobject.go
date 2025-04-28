package distobject

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var ctx = context.Background()

type DistObject struct {
	redis     *RedisClient
	prefix    string
	channel   string
	id        string
	original  map[string]string
	changed   map[string]bool
	objectMap map[string]interface{}
	mu        sync.RWMutex
}

func NewDistObject(r *RedisClient, prefix string, channel string) *DistObject {
	return &DistObject{
		redis:     r,
		prefix:    prefix,
		channel:   channel,
		original:  make(map[string]string),
		changed:   make(map[string]bool),
		objectMap: make(map[string]interface{}),
	}
}

func generateULID() string {
	entropy := ulid.Monotonic(nil, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return id.String()
}

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

		if oldVal, exists := d.original[tag]; !exists || oldVal != value || d.changed[tag] {
			fields[tag] = value
		}
	}

	now := fmt.Sprintf("%d", time.Now().Unix())
	fields["updated_at"] = now
	if _, ok := d.original["created_at"]; !ok {
		fields["created_at"] = now
	}

	_, err := d.redis.Client().HSet(ctx, d.id, fields).Result()
	if err != nil {
		return err
	}

	notif := map[string]interface{}{
		"id":      d.id,
		"changes": fields,
	}
	notifBytes, _ := json.Marshal(notif)
	_ = d.redis.Client().Publish(ctx, d.channel, notifBytes).Err()

	for k, v := range fields {
		d.original[k] = fmt.Sprintf("%v", v)
	}
	d.changed = make(map[string]bool)

	return nil
}

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

func (d *DistObject) MarkChanged(field string) {
	d.changed[field] = true
}

func (d *DistObject) ID() string {
	return d.id
}

func (d *DistObject) AddObject(id string, obj interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.objectMap[id] = obj
}

func (d *DistObject) StartListener() error {
	pubsub := d.redis.Client().Subscribe(ctx, d.channel)

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var payload struct {
				ID      string            `json:"id"`
				Changes map[string]string `json:"changes"`
			}
			err := json.Unmarshal([]byte(msg.Payload), &payload)
			if err != nil {
				continue
			}

			d.mu.RLock()
			obj, exists := d.objectMap[payload.ID]
			d.mu.RUnlock()
			if !exists {
				continue
			}

			v := reflect.ValueOf(obj).Elem()
			t := v.Type()

			for i := 0; i < v.NumField(); i++ {
				field := t.Field(i)
				tag := field.Tag.Get("redis")
				if tag == "" {
					tag = strings.ToLower(field.Name)
				}

				if newVal, ok := payload.Changes[tag]; ok {
					v.Field(i).Set(reflect.ValueOf(newVal))
				}
			}
		}
	}()

	return nil
}
