package crons

import (
	"log"
	"math/big"
	"time"

	"github.com/robfig/cron"
	"github.com/tomochain/tomox-sdk/types"
	"github.com/tomochain/tomox-sdk/utils"
	"github.com/tomochain/tomox-sdk/ws"
)

// tickStreamingCron takes instance of cron.Cron and adds tickStreaming
// crons according to the durations mentioned in config/app.yaml file
func (s *CronService) startPriceBoardCron(c *cron.Cron) {
	c.AddFunc("*/3 * * * * *", s.getPriceBoardData())
}

// tickStream function fetches latest tick based on unit and duration for each pair
// and broadcasts the tick to the client subscribed to pair's respective channel
func (s *CronService) getPriceBoardData() func() {
	return func() {
		pairs, err := s.PairService.GetAll()

		if err != nil {
			log.Println(err.Error())
		}

		for _, p := range pairs {
			bt := p.BaseTokenAddress
			qt := p.QuoteTokenAddress
			p := make([]types.PairAddresses, 0)
			p = []types.PairAddresses{{
				BaseToken:  bt,
				QuoteToken: qt,
			}}

			// Fix the value at 1 day because we only care about 24h change
			duration := int64(1)
			unit := "day"

			ticks, err := s.PriceBoardService.GetPriceBoardData(p, duration, unit)
			if err != nil {
				log.Printf("%s", err)
				return
			}

			quoteToken, err := s.PriceBoardService.TokenDao.GetByAddress(qt)

			if err != nil {
				log.Printf("%s", err)
				return
			}

			var lastTradePrice string
			lastTrade, err := s.PriceBoardService.TradeDao.GetLatestTrade(bt, qt)
			if lastTrade == nil {
				lastTradePrice = "?"
			} else {
				lastTradePrice = lastTrade.PricePoint.String()
			}

			if err != nil {
				log.Printf("%s", err)
				return
			}

			id := utils.GetPriceBoardChannelID(bt, qt)
			var usd *big.Float
			usd, e := s.OHLCVService.GetLastPriceCurrentByTime(quoteToken.Symbol, time.Now())
			if e != nil || usd == nil {
				usd = big.NewFloat(0)
			}
			result := types.PriceBoardData{
				Ticks:          ticks,
				PriceUSD:       usd.String(),
				LastTradePrice: lastTradePrice,
			}

			ws.GetPriceBoardSocket().BroadcastMessage(id, result)
		}
	}
}
