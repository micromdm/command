// Package simple implements command.Service using BoltDB and
// an NSQ Producer as dependencies.
package simple

import (
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/micromdm/mdm"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"

	"github.com/micromdm/command"
)

const (

	// CommandBucket is the *bolt.DB bucket where commands are archived.
	CommandBucket = "mdm.Command.ARCHIVE"

	// CommandTopic is an NSQ topic that events are published to.
	CommandTopic = "mdm.Command"
)

// The publisher interface is satisfied by an NSQ producer.
// Only used in tests.
type publisher interface {
	Publish(string, []byte) error
}

// CommandService creates new MDM Payload and publishes them to an NSQ topic.
// The CommandService also archives all commands to a BoltDB bucket.
type CommandService struct {
	db *bolt.DB
	publisher
}

// NewService creates a CommandService.
func NewService(db *bolt.DB, producer *nsq.Producer) (*CommandService, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(CommandBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &CommandService{db, producer}, nil
}

// NewCommand creates an MDM Payload from an MDM request.
func (svc *CommandService) NewCommand(ctx context.Context, request *mdm.CommandRequest) (*mdm.Payload, error) {
	if request == nil {
		return nil, errors.New("empty CommandRequest")
	}
	payload, err := mdm.NewPayload(request)
	if err != nil {
		return nil, err
	}
	event := command.NewEvent(*payload)
	msg, err := command.MarshalEvent(event)
	if err != nil {
		return nil, err
	}
	if err := svc.archive(event.Time.UnixNano(), msg); err != nil {
		return nil, err
	}
	if err := svc.Publish(CommandTopic, msg); err != nil {
		return nil, err
	}
	return payload, nil
}

// archive events to BoltDB bucket using timestamp as key to preserve order.
func (svc *CommandService) archive(nano int64, msg []byte) error {
	tx, err := svc.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bkt := tx.Bucket([]byte(CommandBucket))
	if bkt == nil {
		return fmt.Errorf("bucket %q not found!", CommandBucket)
	}
	key := []byte(fmt.Sprintf("%d", nano))
	if err := bkt.Put(key, msg); err != nil {
		return err
	}
	return tx.Commit()
}
