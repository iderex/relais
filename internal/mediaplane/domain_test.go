// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package mediaplane

import (
	"reflect"
	"testing"
)

// The states below are the ones issue #31 asks to be named as legal. Each has its
// own test rather than sharing one, because a single test asserting six things
// stops at the first failure and reports one of them.

// TestAnOpenedRoomWithNobodyInItIsLegal covers the zero value, which is the room
// a caller has just opened. Every derivation has to answer on it, because the
// alternative is a caller checking for nil maps before every question.
func TestAnOpenedRoomWithNobodyInItIsLegal(t *testing.T) {
	var room Room

	if !room.IsEmpty() {
		t.Error("a room with no participants reports itself as not empty")
	}
	if room.HasParticipant("nobody") {
		t.Error("a room with no participants claims to hold one")
	}
	if got := room.TracksOf("nobody"); len(got) != 0 {
		t.Errorf("TracksOf on an empty room returned %d tracks, want 0", len(got))
	}
	if got := room.SubscribersOf("nothing"); len(got) != 0 {
		t.Errorf("SubscribersOf on an empty room returned %d subscribers, want 0", len(got))
	}
	if got := room.SubscriptionsOf("nobody"); len(got) != 0 {
		t.Errorf("SubscriptionsOf on an empty room returned %d subscriptions, want 0", len(got))
	}
}

// TestAParticipantPublishingNothingIsLegal is the state every joiner is in for as
// long as it takes them to publish, and for the whole call if they only listen.
func TestAParticipantPublishingNothingIsLegal(t *testing.T) {
	room := Room{
		ID:           "room",
		Participants: map[ParticipantID]Participant{"listener": {ID: "listener"}},
	}

	if !room.HasParticipant("listener") {
		t.Fatal("an admitted participant is not in the room")
	}
	if got := room.TracksOf("listener"); len(got) != 0 {
		t.Errorf("a participant publishing nothing has %d tracks, want 0", len(got))
	}
	if room.IsEmpty() {
		t.Error("a room holding one participant reports itself as empty")
	}
}

// TestATrackNobodySubscribedToIsLegal is the state of every track between the
// moment it appears and the moment somebody asks for it, and the permanent state
// of a track in a room where nobody wants it.
func TestATrackNobodySubscribedToIsLegal(t *testing.T) {
	room := Room{
		ID:           "room",
		Participants: map[ParticipantID]Participant{"speaker": {ID: "speaker"}},
		Tracks: map[TrackID]TrackDescription{
			"t1": {ID: "t1", Publisher: "speaker", Media: MediaAudio, Layers: []LayerID{"a"}},
		},
	}

	if got := room.SubscribersOf("t1"); len(got) != 0 {
		t.Errorf("an unsubscribed track has %d subscribers, want 0", len(got))
	}
	if got := room.TracksOf("speaker"); len(got) != 1 {
		t.Fatalf("the publisher has %d tracks, want 1", len(got))
	}
}

// TestAStoppedTrackReportsNoLayerAndIsStillATrack is the state a track is in
// while its publisher has it stopped. The description stays, the layer list is
// empty, and a subscription to it stays alive, so nothing here may treat an empty
// layer list as an absent track.
func TestAStoppedTrackReportsNoLayerAndIsStillATrack(t *testing.T) {
	stopped := TrackDescription{ID: "t1", Publisher: "speaker", Media: MediaVideo}

	if !stopped.Valid() {
		t.Error("a stopped track is reported as an invalid description")
	}
	if stopped.HasLayer("high") {
		t.Error("a stopped track claims to report a layer")
	}

	room := Room{
		ID:     "room",
		Tracks: map[TrackID]TrackDescription{"t1": stopped},
		Subscriptions: map[SubscriptionID]Subscription{
			"s1": {ID: "s1", Subscriber: "listener", Target: SubscriptionTarget{Track: "t1"}},
		},
	}
	if got := room.SubscribersOf("t1"); len(got) != 1 {
		t.Errorf("a subscription to a stopped track disappeared: %d subscribers, want 1", len(got))
	}
}

// TestASubscriptionReceivingNothingIsLegal is the state between creating a
// subscription and media arriving under it, and the state of every subscription
// to a stopped track.
//
// The model expresses it by holding no such field at all, so the assertion is
// about the shape of the type. See TestOneFactIsHeldInOnePlace for why that is
// the form the guard takes.
func TestASubscriptionReceivingNothingIsLegal(t *testing.T) {
	sub := Subscription{ID: "s1", Subscriber: "listener", Target: SubscriptionTarget{Track: "t1"}}

	if sub.Target.ChoosesLayer() {
		t.Error("a target with no named layer claims to have chosen one")
	}

	room := Room{
		ID:            "room",
		Subscriptions: map[SubscriptionID]Subscription{sub.ID: sub},
	}
	if got := room.SubscriptionsOf("listener"); len(got) != 1 {
		t.Errorf("a subscription receiving nothing is not held: got %d, want 1", len(got))
	}
}

// TestParticipantCarriesNothingButItsIdentifier is the guard the doc comment on
// [Participant] names.
//
// A display name, an address or an account identifier added to this struct is the
// change that puts personal data into this project's logs, metrics and crash
// output on the first join, which docs/decisions/admission.md forbids and nothing
// else in the tree yet refuses. It is a one-line change somebody makes for a good
// local reason, so the refusal is here.
func TestParticipantCarriesNothingButItsIdentifier(t *testing.T) {
	fields := reflect.TypeFor[Participant]().NumField()
	if fields != 1 {
		t.Fatalf("Participant has %d fields, want 1", fields)
	}
	if name := reflect.TypeFor[Participant]().Field(0).Name; name != "ID" {
		t.Errorf("Participant's only field is %q, want \"ID\"", name)
	}
}

// TestOneFactIsHeldInOnePlace is the guard behind the authority rule on [Room]
// and on [Subscription].
//
// Room holds three facts and one identifier, and every other question about a
// room is derived by walking those three. A fourth field is the change that
// creates a second place for a fact that already has one, and the failure it
// produces is the two places disagreeing, which shows up long afterwards as a
// count that is wrong by one.
//
// Subscription is here for the same reason from the other direction: a field
// saying whether media is flowing would be a copy of something the media side
// observes and reports as an event.
func TestOneFactIsHeldInOnePlace(t *testing.T) {
	for _, c := range []struct {
		what  string
		got   int
		want  int
		names []string
	}{
		{"Room", reflect.TypeFor[Room]().NumField(), 4, []string{"ID", "Participants", "Tracks", "Subscriptions"}},
		{"Subscription", reflect.TypeFor[Subscription]().NumField(), 3, []string{"ID", "Subscriber", "Target"}},
	} {
		if c.got != c.want {
			t.Errorf("%s has %d fields, want %d", c.what, c.got, c.want)
			continue
		}
		for i, want := range c.names {
			var typ reflect.Type
			if c.what == "Room" {
				typ = reflect.TypeFor[Room]()
			} else {
				typ = reflect.TypeFor[Subscription]()
			}
			if got := typ.Field(i).Name; got != want {
				t.Errorf("%s field %d is %q, want %q", c.what, i, got, want)
			}
		}
	}
}

// TestADerivationAnswersAboutTheThingItWasAsked is the near miss: every
// derivation walks a map and filters it, and the mistake somebody makes is
// filtering on the wrong side of the comparison, which returns everything.
func TestADerivationAnswersAboutTheThingItWasAsked(t *testing.T) {
	room := Room{
		ID: "room",
		Participants: map[ParticipantID]Participant{
			"speaker": {ID: "speaker"}, "other": {ID: "other"}, "listener": {ID: "listener"},
		},
		Tracks: map[TrackID]TrackDescription{
			"t1": {ID: "t1", Publisher: "speaker", Media: MediaAudio, Layers: []LayerID{"a"}},
			"t2": {ID: "t2", Publisher: "other", Media: MediaVideo, Layers: []LayerID{"low", "high"}},
		},
		Subscriptions: map[SubscriptionID]Subscription{
			"s1": {ID: "s1", Subscriber: "listener", Target: SubscriptionTarget{Track: "t1"}},
			"s2": {ID: "s2", Subscriber: "speaker", Target: SubscriptionTarget{Track: "t2", Layer: "low"}},
		},
	}

	tracks := room.TracksOf("speaker")
	if len(tracks) != 1 || tracks[0].ID != "t1" {
		t.Errorf("TracksOf(speaker) = %v, want exactly t1", tracks)
	}

	subscribers := room.SubscribersOf("t1")
	if len(subscribers) != 1 || subscribers[0] != "listener" {
		t.Errorf("SubscribersOf(t1) = %v, want exactly listener", subscribers)
	}

	subs := room.SubscriptionsOf("speaker")
	if len(subs) != 1 || subs[0].ID != "s2" {
		t.Errorf("SubscriptionsOf(speaker) = %v, want exactly s2", subs)
	}
	if !subs[0].Target.ChoosesLayer() {
		t.Error("a target naming a layer reports that it chose none")
	}
}

// TestADerivationAnswersTheSameWayTwice is the guard behind the ordering in every
// derivation.
//
// Go randomises map iteration on purpose, so a derivation that returns what the
// range produced is a function whose answer changes between two calls on
// unchanged state. The failure that produces is a caller comparing two answers,
// seeing a difference, and acting on a change that did not happen. Removing the
// sort from any derivation reds this.
func TestADerivationAnswersTheSameWayTwice(t *testing.T) {
	room := Room{ID: "room", Tracks: map[TrackID]TrackDescription{}, Subscriptions: map[SubscriptionID]Subscription{}}
	for _, id := range []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8"} {
		room.Tracks[TrackID(id)] = TrackDescription{
			ID: TrackID(id), Publisher: "speaker", Media: MediaVideo, Layers: []LayerID{"low"},
		}
		room.Subscriptions[SubscriptionID("s"+id)] = Subscription{
			ID: SubscriptionID("s" + id), Subscriber: "listener", Target: SubscriptionTarget{Track: TrackID(id)},
		}
	}

	first := room.TracksOf("speaker")
	firstSubs := room.SubscriptionsOf("listener")
	for i := range 40 {
		if got := room.TracksOf("speaker"); !reflect.DeepEqual(got, first) {
			t.Fatalf("TracksOf answered differently on call %d: %v, first %v", i, got, first)
		}
		if got := room.SubscriptionsOf("listener"); !reflect.DeepEqual(got, firstSubs) {
			t.Fatalf("SubscriptionsOf answered differently on call %d: %v, first %v", i, got, firstSubs)
		}
	}
}

// TestOneSubscriberIsReportedOnce covers the deduplication in SubscribersOf. This
// package holds no rule against two subscriptions to one track by one subscriber,
// so the derivation has to answer the question it was asked rather than counting
// subscriptions.
func TestOneSubscriberIsReportedOnce(t *testing.T) {
	room := Room{
		ID: "room",
		Subscriptions: map[SubscriptionID]Subscription{
			"s1": {ID: "s1", Subscriber: "listener", Target: SubscriptionTarget{Track: "t1", Layer: "low"}},
			"s2": {ID: "s2", Subscriber: "listener", Target: SubscriptionTarget{Track: "t1", Layer: "high"}},
		},
	}

	if got := room.SubscribersOf("t1"); len(got) != 1 || got[0] != "listener" {
		t.Errorf("SubscribersOf(t1) = %v, want exactly one listener", got)
	}
}

// TestATrackDescriptionSaysWhetherItWasFilledIn covers the zero value reaching a
// caller that expected a report from the media side.
func TestATrackDescriptionSaysWhetherItWasFilledIn(t *testing.T) {
	for _, c := range []struct {
		what string
		desc TrackDescription
		want bool
	}{
		{"the zero value", TrackDescription{}, false},
		{"no media kind", TrackDescription{ID: "t1", Publisher: "speaker"}, false},
		{"no publisher", TrackDescription{ID: "t1", Media: MediaAudio}, false},
		{"no identifier", TrackDescription{Publisher: "speaker", Media: MediaAudio}, false},
		{"audio with no layer", TrackDescription{ID: "t1", Publisher: "speaker", Media: MediaAudio}, true},
		{"video with layers", TrackDescription{ID: "t1", Publisher: "speaker", Media: MediaVideo, Layers: []LayerID{"low"}}, true},
	} {
		if got := c.desc.Valid(); got != c.want {
			t.Errorf("%s: Valid() = %v, want %v", c.what, got, c.want)
		}
	}
}

// TestOnlyAudioAndVideoAreKinds keeps the zero value out of the set. A kind added
// to the constants without being added here is the change this catches.
func TestOnlyAudioAndVideoAreKinds(t *testing.T) {
	for _, c := range []struct {
		kind  MediaKind
		valid bool
		name  string
	}{
		{MediaUnspecified, false, "unspecified"},
		{MediaAudio, true, "audio"},
		{MediaVideo, true, "video"},
		{MediaKind(200), false, "invalid"},
	} {
		if got := c.kind.Valid(); got != c.valid {
			t.Errorf("%s: Valid() = %v, want %v", c.name, got, c.valid)
		}
		if got := c.kind.String(); got != c.name {
			t.Errorf("String() = %q, want %q", got, c.name)
		}
	}
}

// TestALayerIsAddressedInsideOneTrack covers HasLayer, which is the check behind
// a refused retarget.
func TestALayerIsAddressedInsideOneTrack(t *testing.T) {
	track := TrackDescription{ID: "t1", Publisher: "speaker", Media: MediaVideo, Layers: []LayerID{"low", "high"}}

	if !track.HasLayer("high") {
		t.Error("a track does not report a layer it carries")
	}
	if track.HasLayer("medium") {
		t.Error("a track reports a layer it does not carry")
	}
	if track.HasLayer("") {
		t.Error("the empty layer, which means let the policy choose, is reported as a layer of the track")
	}
}
