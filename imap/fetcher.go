package imap

import (
	"io"
	"log"

	"igcmailimap/config"
	"igcmailimap/state"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Fetcher connects to IMAP over TLS, selects Inbox, and fetches new messages by UID.
type Fetcher struct {
	cfg   *config.Config
	state *state.State
}

// FetchedMessage holds raw RFC822 body for a message (for extraction).
type FetchedMessage struct {
	UID     uint32
	Subject string
	From    string
	Body    []byte
}

// NewFetcher returns a fetcher for the given config and state.
func NewFetcher(cfg *config.Config, st *state.State) *Fetcher {
	return &Fetcher{cfg: cfg, state: st}
}

// FetchNew connects, selects INBOX, fetches messages with UID > state.LastUID, and returns their bodies.
// State is updated to the max UID fetched so each mail is only ever fetched once (last email is not re-processed every poll).
func (f *Fetcher) FetchNew() ([]FetchedMessage, error) {
	if f.cfg.IMAPServer == "" || f.cfg.IMAPUser == "" || f.cfg.IMAPPassword == "" {
		return nil, nil // no config, skip
	}

	c, err := client.DialTLS(f.cfg.IMAPServer, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := c.Logout(); err != nil && err != io.EOF {
			log.Printf("IMAP Logout: %v", err)
		}
	}()

	if err := c.Login(f.cfg.IMAPUser, f.cfg.IMAPPassword); err != nil {
		return nil, err
	}

	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	// Fetch all messages to work around potential IMAP server issues with UID ranges
	seqSet := imap.SeqSet{}
	seqSet.AddRange(1, 0) // 1:*

	// Fetch full message body (RFC822) for the UID range
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchUid, imap.FetchEnvelope}

	ch := make(chan *imap.Message, 10)
	go func() {
		if err := c.UidFetch(&seqSet, items, ch); err != nil {
			log.Printf("UidFetch: %v", err)
		}
	}()

	var out []FetchedMessage
	var fetchedUIDs []uint32
	for msg := range ch {
		if msg == nil {
			continue
		}

		// Skip messages we've already processed
		if msg.Uid <= f.state.LastUID {
			continue
		}

		lit := msg.GetBody(section)
		if lit == nil {
			continue
		}
		body, err := io.ReadAll(lit)
		if err != nil {
			log.Printf("Read body UID %d: %v", msg.Uid, err)
			continue
		}

		// Extract subject and from from envelope
		subject := ""
		from := ""
		if msg.Envelope != nil {
			if msg.Envelope.Subject != "" {
				subject = msg.Envelope.Subject
			}
			if len(msg.Envelope.From) > 0 && msg.Envelope.From[0] != nil {
				from = msg.Envelope.From[0].Address()
			}
		}

		out = append(out, FetchedMessage{
			UID:     msg.Uid,
			Subject: subject,
			From:    from,
			Body:    body,
		})
		fetchedUIDs = append(fetchedUIDs, msg.Uid)
	}

	// Advance state to max UID we fetched so we never re-fetch the same mail (fixes "last email always processed").
	if len(fetchedUIDs) > 0 {
		path, _ := config.StatePath()
		_ = state.UpdateLastUID(path, f.state, fetchedUIDs)
	}
	return out, nil
}

// FetchNewBytes is a convenience that returns bodies as byte slices (same as FetchNew but typo-safe name).
func (f *Fetcher) FetchNewBytes() ([][]byte, []uint32, error) {
	msgs, err := f.FetchNew()
	if err != nil {
		return nil, nil, err
	}
	bodies := make([][]byte, len(msgs))
	uids := make([]uint32, len(msgs))
	for i := range msgs {
		bodies[i] = msgs[i].Body
		uids[i] = msgs[i].UID
	}
	return bodies, uids, nil
}
