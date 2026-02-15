package constructorgen

import (
	"github.com/miyamo2/braider/pkg/annotation"
	"github.com/miyamo2/braider/pkg/annotation/inject"
)

type NotificationService struct { // want "missing constructor for NotificationService"
	annotation.Injectable[inject.Default]
	sender  MessageSender    `braider:"primarySender"`
	tracker ActivityTracker
	storage StorageBackend `braider:"archiveStorage"`
}

type MessageSender interface {
	Send(to, subject, body string) error
}

type ActivityTracker interface {
	Track(event string)
}

type StorageBackend interface {
	Store(key string, data []byte) error
}
