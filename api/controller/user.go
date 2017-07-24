package controller

import (
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
)

func GetUserSettings(database moira.Database, userLogin string) (*dto.UserSettings, *dto.ErrorResponse) {
	userSettings := &dto.UserSettings{
		User:          dto.User{Login: userLogin},
		Contacts:      make([]moira.ContactData, 0),
		Subscriptions: make([]moira.SubscriptionData, 0),
	}

	subscriptionIds, err := database.GetUserSubscriptionIds(userLogin)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	userSettings.Subscriptions, err = database.GetSubscriptions(subscriptionIds)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	contactIds, err := database.GetUserContacts(userLogin)
	if err != nil {
		return nil, dto.ErrorInternalServer(err)
	}

	//todo это нихрена не быстро работает
	for _, id := range contactIds {
		contact, err := database.GetContact(id)
		if err != nil {
			continue
		}
		userSettings.Contacts = append(userSettings.Contacts, contact)
	}

	return userSettings, nil
}