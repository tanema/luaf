warn("@allow") -- unknown control, ignored
warn("@off", "XXX", "@off") -- these are not control messages
warn("@off") -- this one is
warn("@on", "YYY", "@on") -- not control, but warn is off
warn("@off") -- keep it off
warn("@on") -- restart warnings
warn("", "@on") -- again, no control, real warning
warn("@on") -- keep it "started"
warn("Z", "Z", "Z") -- common warning
