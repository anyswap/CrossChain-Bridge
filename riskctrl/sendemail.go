package riskctrl

import (
	"fmt"
	"time"

	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tools"
)

var (
	prevSendAuditEmailTimestamp int64
	minSendAuditEmailInterval   int64 = 1800 // unit seconds
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
	if prevSendAuditEmailTimestamp+minSendAuditEmailInterval > now {
		return nil // too frequently
	}
	prevSendAuditEmailTimestamp = now
	return sendEmail(subject, content, "audit")
}
