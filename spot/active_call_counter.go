package spot

type activeBandCallKey struct {
	band string
	call string
}

// activeBandCallCounter tracks distinct active calls globally and per band.
// Callers own synchronization and must apply add/remove transitions exactly
// once per active key lifecycle.
type activeBandCallCounter struct {
	callRefs   map[string]int
	bandRefs   map[activeBandCallKey]int
	bandCounts map[string]int
	total      int
}

func (c *activeBandCallCounter) Add(call, band string) {
	if call == "" {
		return
	}
	if c.callRefs == nil {
		c.callRefs = make(map[string]int, 64)
	}
	if c.bandRefs == nil {
		c.bandRefs = make(map[activeBandCallKey]int, 64)
	}
	if c.bandCounts == nil {
		c.bandCounts = make(map[string]int, 16)
	}
	if c.callRefs[call] == 0 {
		c.total++
	}
	c.callRefs[call]++
	if band == "" {
		return
	}
	key := activeBandCallKey{band: band, call: call}
	if c.bandRefs[key] == 0 {
		c.bandCounts[band]++
	}
	c.bandRefs[key]++
}

func (c *activeBandCallCounter) Remove(call, band string) {
	if call == "" || c.callRefs == nil {
		return
	}
	refs := c.callRefs[call]
	if refs <= 0 {
		return
	}
	switch {
	case refs == 1:
		delete(c.callRefs, call)
		if c.total > 0 {
			c.total--
		}
	case refs > 1:
		c.callRefs[call] = refs - 1
	}
	if band == "" || c.bandRefs == nil {
		return
	}
	key := activeBandCallKey{band: band, call: call}
	bandRefs := c.bandRefs[key]
	if bandRefs <= 0 {
		return
	}
	switch {
	case bandRefs == 1:
		delete(c.bandRefs, key)
		if c.bandCounts != nil {
			if count := c.bandCounts[band]; count <= 1 {
				delete(c.bandCounts, band)
			} else {
				c.bandCounts[band] = count - 1
			}
		}
	case bandRefs > 1:
		c.bandRefs[key] = bandRefs - 1
	}
}

func (c *activeBandCallCounter) Total() int {
	if c == nil {
		return 0
	}
	return c.total
}

func (c *activeBandCallCounter) CountsByBand() map[string]int {
	if c == nil || len(c.bandCounts) == 0 {
		return nil
	}
	out := make(map[string]int, len(c.bandCounts))
	for band, count := range c.bandCounts {
		if count > 0 {
			out[band] = count
		}
	}
	return out
}
