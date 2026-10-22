// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package apiv2

// Mandarin: /api/v2 routes for push notifications. A new file, like
// household_shopping.go. The logic lives in pkg/household/push.

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/push"

	"github.com/danielgtaylor/huma/v2"
	"xorm.io/xorm"
)

func init() { AddRouteRegistrar(RegisterHouseholdPushRoutes) }

// PushKey is the server's public key, which a device subscribes with.
type PushKey struct {
	PublicKey string `json:"public_key" doc:"The VAPID public key, base64url; pass it as applicationServerKey to PushManager.subscribe."`
}

// PushDevices lists the caller's devices.
type PushDevices struct {
	Devices []*push.Device `json:"devices"`
}

// PushTestResult says how many devices a test push reached.
type PushTestResult struct {
	Sent int `json:"sent" doc:"How many devices the push services accepted it for."`
}

type pushEndpointInput struct {
	Body struct {
		Endpoint string `json:"endpoint" minLength:"1" maxLength:"2000"`
	}
}

// RegisterHouseholdPushRoutes wires push notifications onto the Huma API.
func RegisterHouseholdPushRoutes(api huma.API) {
	tags := []string{"push"}
	Register(api, huma.Operation{
		OperationID: "push-key",
		Summary:     "Get the server's push key",
		Method:      http.MethodGet, Path: "/push/key", Tags: tags,
	}, pushKey)
	Register(api, huma.Operation{
		OperationID: "push-devices-list",
		Summary:     "List my devices with notifications on",
		Method:      http.MethodGet, Path: "/push/devices", Tags: tags,
	}, pushDevicesList)
	Register(api, huma.Operation{
		OperationID: "push-devices-save",
		Summary:     "Turn on notifications for a device",
		Description: "Send the browser's PushSubscription (endpoint and keys) and a name for the list. Sending the same endpoint again updates it; if it belonged to someone else on this device, it's now yours.",
		Method:      http.MethodPost, Path: "/push/devices", Tags: tags,
	}, pushDevicesSave)
	Register(api, huma.Operation{
		OperationID: "push-devices-delete",
		Summary:     "Turn off notifications for one of my devices",
		Method:      http.MethodDelete, Path: "/push/devices/{device}", Tags: tags,
	}, pushDevicesDelete)
	Register(api, huma.Operation{
		OperationID: "push-devices-remove",
		Summary:     "Turn off notifications for this device",
		Description: "By its push endpoint, for the device the app is running on.",
		Method:      http.MethodPost, Path: "/push/devices/remove", Tags: tags,
		DefaultStatus: http.StatusOK,
	}, pushDevicesRemove)
	Register(api, huma.Operation{
		OperationID: "push-test",
		Summary:     "Send a test notification to my devices",
		Method:      http.MethodPost, Path: "/push/test", Tags: tags,
		DefaultStatus: http.StatusOK,
	}, pushTest)
	Register(api, huma.Operation{
		OperationID: "push-settings-get",
		Summary:     "Get my notification preferences",
		Method:      http.MethodGet, Path: "/push/settings", Tags: tags,
	}, pushSettingsGet)
	Register(api, huma.Operation{
		OperationID: "push-settings-update",
		Summary:     "Save my notification preferences",
		Method:      http.MethodPut, Path: "/push/settings", Tags: tags,
	}, pushSettingsUpdate)
}

// inPushSession runs fn in a committed session.
func inPushSession[T any](fn func(s *xorm.Session) (T, error)) (T, error) {
	var zero T
	s := db.NewSession()
	defer s.Close()
	out, err := fn(s)
	if err != nil {
		_ = s.Rollback()
		return zero, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return zero, translateDomainError(err)
	}
	return out, nil
}

func pushKey(ctx context.Context, _ *struct{}) (*singleBody[PushKey], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	key, err := inPushSession(func(s *xorm.Session) (string, error) { return push.PublicKey(s, a) })
	if err != nil {
		return nil, err
	}
	return &singleBody[PushKey]{Body: &PushKey{PublicKey: key}}, nil
}

func pushDevicesList(ctx context.Context, _ *struct{}) (*singleBody[PushDevices], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	devices, err := inPushSession(func(s *xorm.Session) ([]*push.Device, error) { return push.ListDevices(s, a) })
	if err != nil {
		return nil, err
	}
	return &singleBody[PushDevices]{Body: &PushDevices{Devices: devices}}, nil
}

func pushDevicesSave(ctx context.Context, in *struct{ Body push.DeviceInput }) (*singleBody[push.Device], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	d, err := inPushSession(func(s *xorm.Session) (*push.Device, error) { return push.SaveDevice(s, a, &in.Body) })
	if err != nil {
		return nil, err
	}
	return &singleBody[push.Device]{Body: d}, nil
}

func pushDevicesDelete(ctx context.Context, in *struct {
	ID int64 `path:"device"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := inPushSession(func(s *xorm.Session) (struct{}, error) { return struct{}{}, push.DeleteDevice(s, a, in.ID) }); err != nil {
		return nil, err
	}
	return &emptyBody{}, nil
}

func pushDevicesRemove(ctx context.Context, in *pushEndpointInput) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := inPushSession(func(s *xorm.Session) (struct{}, error) {
		return struct{}{}, push.DeleteDeviceByEndpoint(s, a, in.Body.Endpoint)
	}); err != nil {
		return nil, err
	}
	return &emptyBody{}, nil
}

func pushTest(ctx context.Context, _ *struct{}) (*singleBody[PushTestResult], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// Checked like any request, then sent outside a session (Send opens its own).
	if _, err := inPushSession(func(s *xorm.Session) (string, error) { return push.PublicKey(s, a) }); err != nil {
		return nil, err
	}
	sent, err := push.SendTest(ctx, a.GetID())
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[PushTestResult]{Body: &PushTestResult{Sent: sent}}, nil
}

func pushSettingsGet(ctx context.Context, _ *struct{}) (*singleBody[push.Preferences], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	p, err := inPushSession(func(s *xorm.Session) (*push.Preferences, error) { return push.GetPreferences(s, a) })
	if err != nil {
		return nil, err
	}
	return &singleBody[push.Preferences]{Body: p}, nil
}

func pushSettingsUpdate(ctx context.Context, in *struct{ Body push.Preferences }) (*singleBody[push.Preferences], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	p, err := inPushSession(func(s *xorm.Session) (*push.Preferences, error) { return push.SavePreferences(s, a, &in.Body) })
	if err != nil {
		return nil, err
	}
	return &singleBody[push.Preferences]{Body: p}, nil
}
