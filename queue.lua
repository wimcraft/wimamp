-- wimamp play queue, loaded into the mpv daemon by wimamp.
-- user-data/wimamp/queue holds playlist entry ids; when a track ends on its own (or on next), the first
-- queued entry still in the playlist plays instead of the next one. Once the queue drains, the playlist
-- goes on after the track that was playing before the queue took over. The playlist order never changes.
local KEY = "user-data/wimamp/queue"
local eof = false -- the last track ended by itself, so the next start is mpv's own choice
local current -- entry id of the track playing now
local resume -- entry id the playlist continues after once the queue drains
local jumping = false -- the next start is this script's own switch

local function queue() return mp.get_property_native(KEY) or {} end

local function positions() -- entry id -> playlist index
	local pos = {}
	for i, e in ipairs(mp.get_property_native("playlist") or {}) do pos[e.id] = i - 1 end
	return pos
end

-- pop removes the first queued id still in the playlist and returns that entry's index
local function pop()
	local q, pos = queue(), positions()
	while #q > 0 do
		local id = table.remove(q, 1)
		if pos[id] then
			mp.set_property_native(KEY, q)
			return pos[id]
		end
	end
	mp.set_property_native(KEY, q)
end

-- advance picks the index to play after the current track: a queued entry, the one after resume,
-- "none" past the playlist end, or nil to leave it to mpv
local function advance()
	local i = pop()
	if i then
		resume = resume or current
		return i
	end
	local r = resume and positions()[resume]
	resume = nil
	if not r then return nil end
	if r + 1 < mp.get_property_number("playlist-count", 0) then return r + 1 end
	return mp.get_property("loop-playlist") == "no" and "none" or 0
end

local function play(i)
	jumping = i ~= "none" -- stopping starts nothing, so no start to skip
	mp.commandv("playlist-play-index", tostring(i))
end

mp.register_event("end-file", function(e) eof = e.reason == "eof" end)

-- runs before mpv opens the entry it picked; switching here means the wrong track never starts
mp.add_hook("on_before_start_file", 50, function()
	if jumping then jumping = false return end
	if not eof then resume = nil return end -- picked by hand: the playlist goes on from there
	eof = false
	local i = advance()
	if i and i ~= mp.get_property_number("playlist-pos") then play(i) end
end)

-- the last entry ended with tracks still queued: mpv went idle instead of starting one
mp.observe_property("idle-active", "bool", function(_, idle)
	if not idle or not eof then return end
	eof = false
	local i = advance()
	if i and i ~= "none" then play(i) end
end)

-- wimamp next: same choice as a track ending, plain playlist-next otherwise
mp.register_script_message("wimamp-next", function()
	local i = advance()
	if i == nil then
		mp.commandv("playlist-next")
	elseif i ~= "none" then
		play(i)
	end
end)

-- a queued entry that starts for any reason leaves the queue, as do entries gone from the playlist
mp.register_event("start-file", function(e)
	current = e.playlist_entry_id
	local q, pos, out = queue(), positions(), {}
	for _, id in ipairs(q) do
		if id ~= e.playlist_entry_id and pos[id] then out[#out + 1] = id end
	end
	if #out ~= #q then mp.set_property_native(KEY, out) end
end)

mp.set_property_native("user-data/wimamp/queue-script", true)
