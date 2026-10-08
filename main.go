// Command skills links the skills in this repository into your agents and
// sets how visible each one is. Bare `skills` opens the garden.
package main

import "github.com/jorden-parker/skills/internal/cli"

var version = "dev"

func main() { cli.Main("skills", version) }
