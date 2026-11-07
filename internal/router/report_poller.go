package router

import "context"

func (p *Poller) reportOptions(version string, deep bool) (ReportOptions, WebReader) {
	var reader WebReader
	if p.Web != nil {
		reader, _ = p.Web()
	}
	if reader.Host == "" {
		reader.Host = p.gatewayHost()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	hint := ""
	if p.gw != nil {
		hint = p.gw.Location
	}
	rejected := reader.Password != "" && p.webBlocked == webKey(reader) && p.webErr == "wrong_password"
	return ReportOptions{
		Version:          version,
		Host:             reader.Host,
		LocationHint:     hint,
		Deep:             deep,
		User:             reader.User,
		Password:         reader.Password,
		PasswordRejected: rejected,
	}, reader
}

func (p *Poller) noteReport(reader WebReader, rep Report) {
	if rep.Deep.Outcome != outcomeRejected {
		return
	}
	p.mu.Lock()
	p.webBlocked, p.webErr, p.webOK = webKey(reader), "wrong_password", false
	p.mu.Unlock()
	p.logf("ROUTER report: the router rejected the saved password, so the app stopped trying")
}

func (p *Poller) Report(ctx context.Context, version string, deep bool) (Report, error) {
	opts, reader := p.reportOptions(version, deep)
	rep, err := CollectReport(ctx, opts)
	if err != nil {
		return rep, err
	}
	p.noteReport(reader, rep)
	return rep, nil
}
