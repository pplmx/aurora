package models

// Candidate is a struct that represents a candidate in the voting system
type Candidate struct {
	Name    string // The name of the candidate
	Party   string // The party of the candidate
	Program string // The program of the candidate
	Image   string // The image of the candidate
}

// GetName is a method that returns the name of the candidate
func (c *Candidate) GetName() string {
	return c.Name
}

// GetParty is a method that returns the party of the candidate
func (c *Candidate) GetParty() string {
	return c.Party
}

// GetProgram is a method that returns the program of the candidate
func (c *Candidate) GetProgram() string {
	return c.Program
}

// GetImage is a method that returns the image of the candidate
func (c *Candidate) GetImage() string {
	return c.Image
}
