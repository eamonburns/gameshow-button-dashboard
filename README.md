# Game Show Button Dashboard

This is a simple TUI application to manage game show buttons similarly to
Jeopardy.

There are three phases:
- [Reading](#reading) the clue
- [Waiting](#waiting) for a buzzer press
- [Answering](#answering) the clue

## Reading

This is for when the game show host is reading a clue. No buzzer presses are
detected during this time.

The host can indicate that they have stopped reading by pressing
<kbd>space</kbd>.

After the host has stopped reading, the program will go to ["waiting"](#waiting).

## Waiting

This is when contestants have a chance to "buzz-in". The host is able to
manually override the buzz-in by pressing a number key that corresponds with a
player's button ID.

Another way to buzz-in is by [receiving a webhook](#webhook).

The host can also choose to exit this page by pressing <kbd>escape</kbd>, and
they will be taken back to the ["reading"](#reading) page.

Once a buzz-in has been detected, the program will go to ["answering"](#answering).

## Answering

This is when the contestant that just buzzed-in has a chance to answer the clue.

The host can mark the currently-buzzed-in contestant's answer as correct by
pressing <kbd>enter</kbd>, or as incorrect by pressing <kbd>backspace</kbd>.

There is a timer that will count down, with a timeout that is
[configurable](#configuration).

The timer is for informational purposes only, and will not automatically mark
the contestant's answer as correct or incorrect. The host is free to do that
even after the timer expires (and must, to progress the game).

Once a contestant's answer is marked as:
- Correct: The program will go back to [reading](#reading)
- Incorrect: The program will go back to [waiting](#waiting), unless all
  contestants have already answered incorrectly. In that case, it will go back
  to [reading](#reading).

# Setup

Requirements:
- [Go v1.25.8+](https://go.dev/doc/install)
    - Check which version you have by running `go version`

Clone the repository:
```sh
git clone https://github.com/eamonburns/gameshow-button-dashboard
cd gameshow-button-dashboard
```

Copy the configuration file, and edit it as needed (see [Configuration](#configuration) for details):
```sh
cp example-config.json config.json
```

Run the program, providing a webhook ID:
```sh
WEBHOOK_ID=my_secret_id go run .
```

## Webhook

The main way that button presses are detected is through webhooks.

While the program is [waiting](#waiting) for a contestant to buzz-in, any HTTP
`POST` request sent to `http://HOST:8080/webhook/WEBHOOK_ID`, with the JSON
payload `{"button_id":ID}` will cause the player with the button ID `ID` to
buzz-in (if the player hasn't buzzed-in already that round).

Example (just using `curl`):
```sh
WEBHOOK_ID=my_secret_id
curl -X POST "http://localhost:8080/webhook/$WEBHOOK_ID" -d '{"button_id":1}'
```

For a program that works for this out-of-the-box, see
[eamonburns/esp32_webhook](https://github.com/eamonburns/esp32_webhook).

## Configuration

The program is configured using a file `config.json` in the current working
directory.

It has the following configuration options:

### `players`

A list of players. Each player has two fields:
- `name`: (string) Name of player
- `button_id`: (integer) Button ID

Both fields must be unique for all players.

### `answer_timeout_seconds`

The number of seconds to give a player to answer the clue.

# Troubleshooting

The program creates a log file `buttons.log` in the current working directory.

The error messages could use some work: #2
