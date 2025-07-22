package terminalparser

type Cursor struct {
	X, Y int
	Hide bool
}

func (c *Cursor) MoveHome() {
	c.X = 1
	c.Y = 1
}

func (c *Cursor) MoveUp(ps int) {
	c.Y -= ps
	if c.Y < 0 {
		c.Y = 0
	}
}

func (c *Cursor) MoveDown(ps int) {
	c.Y += ps
}

func (c *Cursor) MoveRight(ps int) {
	c.X += ps
}

func (c *Cursor) MoveLeft(ps int) {
	c.X -= ps
	if c.X < 0 {
		c.X = 1
	}
}
