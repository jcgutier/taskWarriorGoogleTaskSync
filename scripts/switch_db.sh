#!/usr/bin/env bash

cp ~/Dropbox/.taskwarrior/taskchampion_bkp.sqlite3 ~/Dropbox/.taskwarrior/taskchampion_bk1.sqlite3
cp ~/Dropbox/.taskwarrior/taskchampion.sqlite3 ~/Dropbox/.taskwarrior/taskchampion_bkp.sqlite3
mv ~/Dropbox/.taskwarrior/taskchampion_bk1.sqlite3 ~/Dropbox/.taskwarrior/taskchampion.sqlite3
