package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

var client *resty.Client
var targetPrice float64
var lastPrice float64 = 0

func init() {
	var err error
	if err = godotenv.Load(); err != nil {
		logrus.WithError(err).Fatal("Error loading .env file")
	}

	targetPrice, err = strconv.ParseFloat(os.Getenv("BELOW_PRICE"), 64)
	if err != nil {
		logrus.WithError(err).WithField("price", os.Getenv("BELOW_PRICE")).Fatal("Invalid target price")
	}

	client = resty.New().SetBaseURL("https://api.opensea.io/api/v1/collection/")

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
}

func main() {
	for {
		time.Sleep(time.Second * 3)
		res, err := client.R().Get(os.Getenv("COLLECTION_ID"))
		if err != nil {
			logrus.WithError(err).Error("Error getting data from OpenSea")
			continue
		}

		var data OpenSeaCollectionData
		err = json.Unmarshal(res.Body(), &data)
		if err != nil {
			logrus.WithError(err).Error("Error unmarshalling data from OpenSea")
			continue
		}

		priceNow := data.Collection.Stats.FloorPrice
		logrus.Infof("Floor price is now %.4f\n", priceNow)

		if lastPrice != priceNow && priceNow <= targetPrice {
			lastPrice = priceNow

			logrus.Infof("Target price reached: %.4f\n", priceNow)
			if err := sendMailFloorAlert(priceNow); err != nil {
				logrus.WithError(err).Error("Error sending mail")
			}
		}
	}
}

type OpenSeaCollectionData struct {
	Collection struct {
		Stats struct {
			FloorPrice float64 `json:"floor_price"`
		} `json:"stats"`
	} `json:"collection"`
}

func sendMailFloorAlert(price float64) error {
	from := mail.NewEmail("Floor Alert", os.Getenv("SENDGRID_FROM_EMAIL"))
	subject := "Floor price is now " + strconv.FormatFloat(price, 'f', 4, 64)
	to := mail.NewEmail("Target user", os.Getenv("ALERT_EMAIL"))
	message := mail.NewSingleEmail(from, subject, to, subject, subject)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	_, err := client.Send(message)

	return err
}
