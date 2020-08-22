package riskctrl

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tools"
)

var (
	prevSendAuditTimestamp int64
	minSendAuditInterval   int64 = 1800 // unit seconds

	prevSendLowReserveTimestamp int64
	minSendLowReserveInterval   int64 = 3600 // unit seconds
)

func sendEmail(subject, content, topic string) error {
	to := riskConfig.Email.To
	cc := riskConfig.Email.Cc
	err := tools.SendEmail(to, cc, subject, content)
	if err != nil {
		log.Error(fmt.Sprintf("[%v] send email failed", topic), "subject", subject, "err", err)
	} else {
		log.Info(fmt.Sprintf("[%v] send email success", topic), "subject", subject)
	}
	return err
}

func sendAuditEmail(subject, content string) error {
	if riskConfig.Email == nil {
		return nil
	}
	now := time.Now().Unix()
	if prevSendAuditTimestamp+minSendAuditInterval > now {
		return nil // too frequently
	}
	prevSendAuditTimestamp = now
	return sendEmail(subject, content, "balance deviation")
}

func sendLowReserveEmail(subject, content string) error {
	if riskConfig.Email == nil {
		return nil
	}
	now := time.Now().Unix()
	if prevSendLowReserveTimestamp+minSendLowReserveInterval > now {
		return nil // too frequently
	}
	prevSendLowReserveTimestamp = now
	return sendEmail(subject, content, "low reserve")
}
