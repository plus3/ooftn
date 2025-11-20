package ecs

type UpdateFrame struct {
	DeltaTime float64
	Commands  *Commands
}

func newUpdateFrame(dt float64, storage *Storage) *UpdateFrame {
	return &UpdateFrame{
		DeltaTime: dt,
		Commands:  newCommands(storage),
	}
}
