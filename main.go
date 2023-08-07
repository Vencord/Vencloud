// Vencord Cloud, the API for the Vencord client mod
// Copyright (C) 2023 Vendicated and contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"os"
	"github.com/gofiber/fiber/v2"
	"github.com/Vencord/Backend/Routers"
)

func main() {
	// environment
	HOST := os.Getenv("HOST")
	PORT := os.Getenv("PORT")
	
	app := fiber.New()

	Routers.Initalize(app)

	app.Listen(HOST + ":" + PORT)
}
