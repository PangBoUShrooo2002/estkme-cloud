package lpa

import (
	"context"
	"log/slog"

	"github.com/damonto/estkme-cloud/internal/config"
	"github.com/damonto/estkme-cloud/internal/driver"
	"github.com/damonto/libeuicc-go"
)

type LPA struct {
	*libeuicc.Libeuicc
}

func New(driver driver.APDU) (*LPA, error) {
	logLevel := libeuicc.LogInfoLevel
	if config.C.Verbose {
		logLevel = libeuicc.LogDebugLevel
	}
	libeuicc, err := libeuicc.New(driver, libeuicc.NewDefaultLogger(logLevel))
	if err != nil {
		return nil, err
	}
	return &LPA{
		Libeuicc: libeuicc,
	}, nil
}

func (l *LPA) Download(ctx context.Context, activationCode *libeuicc.ActivationCode, downloadOption *libeuicc.DownloadOption) error {
	return l.sendNotificationsAfterExecution(func() error {
		return l.DownloadProfile(ctx, activationCode, downloadOption)
	})
}

func (l *LPA) Delete(iccid string) error {
	return l.sendNotificationsAfterExecution(func() error {
		return l.DeleteProfile(iccid)
	})
}

func (l *LPA) sendNotificationsAfterExecution(action func() error) error {
	currentNotifications, err := l.GetNotifications()
	if err != nil {
		return err
	}
	lastSeqNumber := currentNotifications[len(currentNotifications)-1].SeqNumber

	if err := action(); err != nil {
		slog.Error("failed to execute action", "error", err)
		return err
	}

	notifications, err := l.GetNotifications()
	if err != nil {
		slog.Error("failed to get notifications", "error", err)
		return err
	}
	for _, notification := range notifications {
		if notification.SeqNumber > lastSeqNumber {
			if err := l.ProcessNotification(
				notification.SeqNumber,
				notification.ProfileManagementOperation != libeuicc.NotificationProfileManagementOperationDelete,
			); err != nil {
				slog.Error("failed to process notification", "error", err, "notification", notification)
				return err
			}
			slog.Info("notification processed", "notification", notification)
		}
	}
	return nil
}

func (l *LPA) FindProfile(iccid string) (*libeuicc.Profile, error) {
	profiles, err := l.GetProfiles()
	if err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		if profile.Iccid == iccid {
			return profile, nil
		}
	}
	return nil, nil
}
