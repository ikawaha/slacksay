slacksay
===

Convert slack messages to audible speech on Mac.

## Install

```
go get github.com/ikawaha/slacksay/...
```

## Usage

### 1. Get your slack token

see. https://api.slack.com/custom-integrations/legacy-tokens or https://api.slack.com/apps

### 2. Run slacksay

```
slacksay -t <slack_token> [-d (<json data>|@<file_name>|@-)]
  -d string
    	json data. If you start the data with the letter @, the rest should be a file name to read the data from, or -  if you  want to read the data from stdin.
  -t string
    	slack token
```

**ex.**

```
slacksay -t xoxp-your-token -d '{"bot_message": "true"}'
```


## Configuration

note. If you do not specify anything for these options, the default values are used.

|Option|default|Description|Example|
|:---|:---|:---|:---|
|command|say|your speech tool(default say)| say|
|channel::yomi| ---| specify the keyword reading| ["random", "ザツダン"]|
|channel::includes| ---| target channels| ["general", "develop"]|
|channel::excludes| ---| ignored channels| ["random"]|
|user::yomi| ---| specify the user reading| ["yamada", "ヤマダ"]|
|user::includes| ---| target users| ["general", "develop"]|
|user::excludes| ---|ignored users| ["random"]|
|keyword::yomi| ---| specify the keyword reading| ["dev", "デブ"]|
|keyword::includes| ---| speak only messages that contain these keywords| ["general", "develop"]|
|keyword::excludes| ---| ignore messages that contain these keywords| ["random"]|
|bot_message| false | ignore bot message if false| true|
|timeout|1m| speech command timeout| 3m10s|

### Example
```json
{
  "command": "say",
  "channel": {
    "yomi": ["random", "ザツダン"],
    "includes": [],
    "excludes": ["bot-report"]
  },
  "user": {
    "yomi": ["yamada", "ヤマダ"],
    "includes": [],
    "excludes": ["my_slack_name"]
  },
  "keyword": {
    "yomi": ["dev", "デブ"],
    "includes": ["レポート"],
    "excludes": ["info:"]
  },
  "bot_message": false,
  "timeout": "30s"
}
```

### Ordr of Filtering

Options are specified, messages are filtered in the following order:

1. channel
   1. includes ?
1. user
   1. includes ?
1. keyword
   1. includes ?
1. bot_message ?
1. channel
   1. excludes ?
1. user
   1. excludes ?
1. keyword
   1. excludes ?

---
MIT
