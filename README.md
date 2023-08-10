# RandomListSampler

Connects to Mailchimp account and tags a random sample of emails on it.

### How to use it ###

This is a command-line tool. Go to the binaries directory and download the right one for your architecture...

- Windows: for 64-bit (most standard machines)  <binaries/randomsampler-amd64.exe> or if on a 32-bit processor <binaries/randomsampler-windows-386.exe>
- Mac: for 64-bit arch, this should work <binaries/randomsampler-darwin-amd64>

Email me at [eric at gmail dot com] if you need an executable that isn't here arleady and you don't have the tools to build one

### Running ###

(Todo: make a video of it running)

Run the file in your command-line ( [Command-line in Windows](https://www.howtogeek.com/235101/10-ways-to-open-the-command-prompt-in-windows-10/)  | [Command-line on Mac](https://www.alphr.com/open-command-prompt-mac/))

The script will guide you from there, *you'll need a Mailchimp account and an API key, which the program will link you to the Mailchimp page for creating*

Exit with ctrl+C or command+C at any time if you want to stop executing the tool.

### Building from code ###

This is written in Golang, so you'll need Golang installed and running. I use a Mac and only have a build script for that, so use build.sh on a Mac and you should be set.

### Why? ###
I wanted to learn Go and had heard a customer ask for something like this, so here it is.
