// Package mediaplane declares the port between everything that orchestrates and
// everything that holds a packet, together with the identifiers and values that
// cross it.
//
// This file holds the domain vocabulary: room, participant, track, layer and
// subscription. Each one has exactly one definition, here, with its lifetime and
// its identity rule beside it, so that three packages cannot end up meaning
// slightly different things by the same word.
//
// The words match docs/decisions/media-plane-port.md and docs/api/contract.md.
// Where this file and either of those disagree, the records are what have to be
// argued with first, and this file is corrected to them rather than the other way
// round.
//
// Nothing here is imported from the transport library and nothing here is
// imported from elsewhere in this repository, which is the property
// docs/architecture.md fixes for this package.
package mediaplane

import (
	"slices"
	"strings"
)

// RoomID names a room. It is minted above the port when the room is opened, so a
// caller can address a room before this project has answered, and it is carried
// downward from then on. It is opaque below the port: nothing there parses it or
// assumes it means anything.
//
// Identity: a room is the room with this identifier, for as long as that room is
// open. The identifier is unique among the rooms open on a host and it is not
// unique over time. A room opened under an identifier a closed room used is a
// different room that happens to share a name, and this package deliberately
// holds no way to tell the two apart, because a room holds no memory of who was
// in it and a way to tell them apart would be that memory.
type RoomID string

// ParticipantID names one connected session. It is minted above the port and is
// opaque here, which docs/decisions/admission.md fixes: it is not resolved, not
// parsed, and not assumed to mean anything outside the service that issued it.
//
// Identity: a participant is the participant with this identifier inside one
// room. It is unique within that room and says nothing across rooms. It is not
// unique over time either: a participant that detaches and is admitted again
// under the same identifier is a new participant, and nothing here says the two
// were the same person, because nothing here knows what a person is.
type ParticipantID string

// TrackID names one media stream from one participant. It is the only identifier
// that is minted below the port, because a track names something only the media
// side can observe, and it is carried upward.
//
// Identity: a track is the track with this identifier for as long as that track
// exists, unique across the host while it does. A track ends when its publisher
// stops publishing it or leaves, and the identifier is not reused to mean
// anything else afterwards.
type TrackID string

// LayerID names one of the alternatives a subscriber can be given for a track. It
// is minted below the port as part of the track's reported structure.
//
// A layer is not a resource. It is part of a track's description, and a caller
// addresses one only inside a [SubscriptionTarget]. What attributes a layer
// carries is fixed by docs/decisions/media-formats.md and by issue #34, and this
// package does not restate them: a second copy of that list here would drift
// against the record that decides it.
//
// Identity: a layer is the layer with this identifier within one track
// description. Two tracks may use the same layer identifier for unrelated things,
// and a comparison of layer identifiers across tracks is a defect.
type LayerID string

// SubscriptionID names one subscriber receiving one track. It is minted above the
// port so the caller can address the subscription before the media side has
// confirmed anything.
//
// Identity: a subscription is the subscription with this identifier inside one
// room, unique within that room. It exists from the moment it is created until it
// is ended, until the track ends, or until either side leaves. Whether media is
// flowing under it is a different question, answered below.
type SubscriptionID string

// MediaKind says whether a track carries audio or video. It is the whole of what
// a track description says about the shape of the media, and it is not the format:
// a subscriber deciding whether it can decode a track needs the format, which
// issue #141 carries into the API contract and which is not modelled here yet.
type MediaKind uint8

// The kinds. MediaUnspecified is the zero value and is not a kind: a track
// description carrying it has not been filled in, which [TrackDescription.Valid]
// is what refuses.
const (
	MediaUnspecified MediaKind = iota
	MediaAudio
	MediaVideo
)

// String names the kind for a log line or a test failure.
func (k MediaKind) String() string {
	switch k {
	case MediaAudio:
		return "audio"
	case MediaVideo:
		return "video"
	case MediaUnspecified:
		return "unspecified"
	default:
		return "invalid"
	}
}

// Valid reports whether the kind is one this project carries.
func (k MediaKind) Valid() bool {
	return k == MediaAudio || k == MediaVideo
}

// Participant is one connected session, and it is its identifier and nothing
// else.
//
// The struct is deliberately one field wide. Everything a service above this one
// knows about the person behind the session stays there, which is what
// docs/decisions/seam.md fixes, and a field added here is a change to that record
// rather than a convenience. TestParticipantCarriesNothingButItsIdentifier is
// what refuses one.
type Participant struct {
	ID ParticipantID
}

// TrackDescription is what the media side reports about one track: its
// identifier, which participant published it, whether it is audio or video, and
// the layers currently available.
//
// The media side is authoritative for every field. A copy of any of them kept
// above the port is a second place to hold one fact, which is the thing this
// model exists to prevent, so [Room] holds the description it was given and does
// not edit it.
type TrackDescription struct {
	ID        TrackID
	Publisher ParticipantID
	Media     MediaKind
	// Layers is the set currently available, which changes over the life of the
	// track. It is legal for it to be empty: a stopped track reports no layer
	// while it is stopped, and a subscription to it stays alive.
	Layers []LayerID
}

// Valid reports whether the description is one the media side could have
// produced. It refuses an unset media kind, an empty track identifier and an
// empty publisher, which are the three ways a zero value reaches a caller that
// expected a report.
func (t TrackDescription) Valid() bool {
	return t.ID != "" && t.Publisher != "" && t.Media.Valid()
}

// HasLayer reports whether the track currently reports the named layer.
//
// It is the check behind a retarget: a target naming a layer a track does not
// report is refused, and the subscription keeps the target it had.
func (t TrackDescription) HasLayer(id LayerID) bool {
	return slices.Contains(t.Layers, id)
}

// SubscriptionTarget is a track identifier and either a named layer or the
// instruction to let the allocation policy choose.
//
// The empty layer is the second of those. It is not a missing value: a caller
// that does not want to choose says so by leaving it empty, which is what
// [SubscriptionTarget.ChoosesLayer] reads.
type SubscriptionTarget struct {
	Track TrackID
	Layer LayerID
}

// ChoosesLayer reports whether the caller named a layer rather than leaving the
// choice to the allocation policy. Which layer the policy then picks is issue
// #36 and is not decided here.
func (t SubscriptionTarget) ChoosesLayer() bool {
	return t.Layer != ""
}

// Subscription connects one subscriber to one track at one target.
//
// Whether media is flowing under it is not a field. That fact is the media side's
// and it arrives as an event, so holding it here would be a copy that has to be
// kept in step with the thing it copies. A subscription that exists and is
// receiving nothing is a normal state and not a degenerate one.
type Subscription struct {
	ID         SubscriptionID
	Subscriber ParticipantID
	Target     SubscriptionTarget
}

// Room is the container everything else hangs on, and the one place each fact
// below is held.
//
// What is authoritative where is the decision this type carries. Participants and
// Subscriptions are the layer above the port speaking: it admitted them and it
// created them, so it holds them. Tracks is the media side speaking: it observed
// them, so what is held here is the last description it reported and nothing
// above the port edits one.
//
// Every other question about a room is answered by walking those three rather
// than by a fourth field. There is no list of a participant's tracks, no count of
// a track's subscribers and no set of a room's layers, because each of those
// would be a second place holding a fact that already has one, and two places for
// one fact is what this model exists to prevent.
//
// The zero value is a legal room: one that has just been opened, with nobody in
// it and nothing published. Every method below works on it.
type Room struct {
	ID            RoomID
	Participants  map[ParticipantID]Participant
	Tracks        map[TrackID]TrackDescription
	Subscriptions map[SubscriptionID]Subscription
}

// HasParticipant reports whether the participant is admitted to this room.
func (r Room) HasParticipant(id ParticipantID) bool {
	_, ok := r.Participants[id]
	return ok
}

// TracksOf returns the tracks the participant is currently publishing, derived
// from what the media side reported rather than from a list kept beside the
// participant.
//
// A participant publishing nothing is a normal state, so the result is empty
// rather than an error. The order is by track identifier, because a map's order
// is not one and a caller comparing two answers needs them to be comparable.
func (r Room) TracksOf(id ParticipantID) []TrackDescription {
	out := make([]TrackDescription, 0, len(r.Tracks))
	for _, t := range r.Tracks {
		if t.Publisher == id {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b TrackDescription) int {
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return out
}

// SubscribersOf returns the participants subscribed to the track, derived from
// the subscriptions rather than from a count kept beside the track.
//
// A track with no subscriber is a normal state: a publisher whose stream nobody
// has asked for is not an error, and code that treats it as one is wrong in
// production. The order is by participant identifier, and the result is
// deduplicated: this package holds no rule against one subscriber holding two
// subscriptions to one track, because that rule belongs to docs/api/contract.md,
// and a derivation that reported such a subscriber twice would be answering a
// question nobody asked.
func (r Room) SubscribersOf(id TrackID) []ParticipantID {
	out := make([]ParticipantID, 0, len(r.Subscriptions))
	for _, s := range r.Subscriptions {
		if s.Target.Track == id {
			out = append(out, s.Subscriber)
		}
	}
	slices.SortFunc(out, func(a, b ParticipantID) int {
		return strings.Compare(string(a), string(b))
	})
	return slices.Compact(out)
}

// SubscriptionsOf returns the subscriptions the participant holds, ordered by
// subscription identifier.
func (r Room) SubscriptionsOf(id ParticipantID) []Subscription {
	out := make([]Subscription, 0, len(r.Subscriptions))
	for _, s := range r.Subscriptions {
		if s.Subscriber == id {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Subscription) int {
		return strings.Compare(string(a.ID), string(b.ID))
	})
	return out
}

// IsEmpty reports whether the room has nobody in it.
//
// It exists so that a caller asking the question does not reach for
// len(r.Participants) == 0 and quietly disagree with the next caller about
// whether a room holding only a stale track counts. An empty room is a legal
// room and closing it is a decision made above this package.
func (r Room) IsEmpty() bool {
	return len(r.Participants) == 0
}
