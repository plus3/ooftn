package ecs

type UpdateFrame struct {
	DeltaTime float64
	Commands  *Commands
	Storage   *Storage
}

func newUpdateFrame(dt float64, storage *Storage) *UpdateFrame {
	return &UpdateFrame{
		DeltaTime: dt,
		Commands:  newCommands(),
		Storage:   storage,
	}
}
