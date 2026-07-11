package job

func (su *SessionUpdater) TriggerNow() {
	su.updateAllSessions()
}
