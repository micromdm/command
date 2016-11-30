package command

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/micromdm/mdm"
	uuid "github.com/satori/go.uuid"

	"github.com/micromdm/command/internal/commandproto"
)

type Event struct {
	ID      string
	Time    time.Time
	Payload mdm.Payload
}

// NewEvent returns an Event with a unique ID and the current time.
func NewEvent(cmd mdm.Payload) *Event {
	event := Event{
		ID:      uuid.NewV4().String(),
		Time:    time.Now().UTC(),
		Payload: cmd,
	}
	return &event
}

// MarshalEvent serializes an event to a protocol buffer wire format.
func MarshalEvent(e *Event) ([]byte, error) {
	payload := &commandproto.Payload{
		CommandUuid: e.Payload.CommandUUID,
	}
	if e.Payload.Command != nil {
		payload.Command = &commandproto.Command{
			RequestType: e.Payload.Command.RequestType,
		}
	}
	switch e.Payload.Command.RequestType {
	case "DeviceInformation":
		payload.Command.DeviceInformation = &commandproto.DeviceInformation{
			Queries: e.Payload.Command.DeviceInformation.Queries,
		}
	case "InstallProfile":
		payload.Command.InstallProfile = &commandproto.InstallProfile{
			Payload: e.Payload.Command.InstallProfile.Payload,
		}
	}
	return proto.Marshal(&commandproto.Event{
		Id:      e.ID,
		Time:    e.Time.UnixNano(),
		Payload: payload,
	})

}

// UnmarshalEvent parses a protocol buffer representation of data into
// the Event.
func UnmarshalEvent(data []byte, e *Event) error {
	var pb commandproto.Event
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	e.ID = pb.Id
	e.Time = time.Unix(0, pb.Time).UTC()
	if pb.Payload == nil {
		return nil
	}
	e.Payload = mdm.Payload{
		CommandUUID: pb.Payload.CommandUuid,
	}
	if pb.Payload.Command == nil {
		return nil
	}
	e.Payload.Command = &mdm.Command{
		RequestType: pb.Payload.Command.RequestType,
	}
	switch pb.Payload.Command.RequestType {
	case "DeviceInformation":
		e.Payload.Command.DeviceInformation = mdm.DeviceInformation{
			Queries: pb.Payload.Command.DeviceInformation.Queries,
		}
	case "InstallProfile":
		e.Payload.Command.InstallProfile = mdm.InstallProfile{
			Payload: pb.Payload.Command.InstallProfile.Payload,
		}
	}
	return nil
}
