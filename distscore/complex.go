package distscore

type UnifyRecord struct {
	UnifyKey
	UnifyVal
}

type UnifyKey struct {
	Source Location
	Server bool
}

type UnifyVal struct {
	Target Location
}

type complexUnifier struct {
	orig LocationUnifier
	recs map[UnifyKey]UnifyVal
}

func NewComplexUnifier(orig LocationUnifier, records []UnifyRecord) LocationUnifier {
	recs := make(map[UnifyKey]UnifyVal)
	for _, rec := range records {
		recs[rec.UnifyKey] = rec.UnifyVal
	}
	return &complexUnifier{
		orig: orig,
		recs: recs,
	}
}

func (u *complexUnifier) Unify(l Location, server bool) Location {
	key := UnifyKey{Source: l, Server: server}
	if val, ok := u.recs[key]; ok {
		return val.Target
	}
	return u.orig.Unify(l, server)
}

func (u *complexUnifier) IsDeputy(l Location) bool {
	return u.orig.IsDeputy(l)
}

type ScoreRecord struct {
	ScoreKey
	ScoreVal
}

type ScoreKey struct {
	Client Location
	Server Location
}

type ScoreVal struct {
	Score float32
	Local bool
}

type complexScorer struct {
	orig DistScorer
	recs map[ScoreKey]ScoreVal
}

func NewComplexScorer(orig DistScorer, records []ScoreRecord) DistScorer {
	recs := make(map[ScoreKey]ScoreVal)
	for _, rec := range records {
		recs[rec.ScoreKey] = rec.ScoreVal
	}
	return &complexScorer{
		orig: orig,
		recs: recs,
	}
}

func (s *complexScorer) DistScore(client, server Location) (score float32, local bool) {
	key := ScoreKey{Client: client, Server: server}
	if val, ok := s.recs[key]; ok {
		return val.Score, val.Local
	}
	return s.orig.DistScore(client, server)
}
