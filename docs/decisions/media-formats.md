# The media formats this project carries

This project forwards and never converts. A packet leaves in the format it
arrived in, and the service holds no encoder and no decoder for the media it
carries.

Recorded for issue #118. It is a property the rest of the tree is allowed to rely
on, and an issue that needs it to be false is changing this record rather than
making an exception inside a forwarding loop.

## Why the position is worth a record

Three reasons, in the order they carry weight.

Conversion is the largest cost a media server can take on. The cost of a forwarded
stream stays close to flat only while nothing decodes it, so a budget measured with
conversion switched on describes a different product from the one this repository
is selling. The resource budget in issue #10 is one of the two things this project
offers, and it would not survive.

Holding no decoder removes the most attacked parsing surface in this class of
software from a service that already takes bytes from strangers. A media decoder is
a large parser handling hostile input, and the ones in wide use have a long
vulnerability history. Not having one is a security property that no amount of
careful coding buys.

A project that converts has to keep choosing which formats to produce, for which
subscriber, at which quality, and that is a maintenance promise with no end in it.

## What this project still has to understand

Not nothing, and this is the part the position is usually stated without.

Enough of each payload format to find frame boundaries, so a packet can be
attributed to the frame it belongs to and a partial frame can be recognised as
partial.

Enough to recognise a point at which a subscriber can start decoding, because a
joiner has to be given a starting point and the keyframe policy in issue #35 is
what decides when one is asked for.

Enough to read the layer structure the selection policy acts on, because issue #34
chooses between the alternatives a publisher offered and issue #36 decides what is
given up first when there is not enough bandwidth.

None of that decodes anything. It reads the headers a payload format puts in front
of its own bytes, and it stops there. Where the transport library already parses
that header, this project reads the library's result rather than parsing it a
second time, and the port record is what keeps the library's own types from
crossing into the orchestration layer.

## The formats, and what is read out of each

This list is what the forwarding core is paid for. Each entry costs code that
tracks a format's rules, so the list is short on purpose and grows by a change to
this record.

Opus, for audio. What is read: whether the packet carries speech at all, so the
audio policy in issue #37 can decide which streams a subscriber receives, and the
packet's place in the stream. Audio has no layer structure to select from here and
no keyframe, which is why the audio entry is the cheapest one.

VP8, for video. What is read: the payload descriptor in front of each packet, which
gives the frame boundary, whether the frame can be decoded without an earlier one,
and the temporal layer the frame belongs to.

VP9, for video. What is read: the payload descriptor, which gives the same three
things plus the spatial layer, so that both dimensions of the layer structure are
visible to the selection policy.

AV1, for video. What is read: the dependency structure the format publishes
alongside the packets, which states which frames a given frame needs. It is read
rather than inferred, because inferring it is where a forwarder starts holding
format knowledge it cannot defend.

H.264, for video. What is read: the packetisation the format defines, enough to
find frame boundaries across the aggregation and fragmentation forms, and enough to
recognise a frame a subscriber can start from.

## What happens to a format not on that list

It is refused, at negotiation, before any packet arrives.

Forwarded blind is the other answer and it is rejected. A forwarder that cannot
find a frame boundary cannot tell a joiner when to start, cannot request a starting
point, and cannot drop a frame without corrupting the ones after it. What that
produces is not a lower quality stream. It is a subscriber who receives bytes and
sees nothing, with no diagnostic, which is exactly the failure mode this project
exists to stop producing.

Refusing at negotiation means the publishing client is told at the moment it offers,
in a place where it can offer something else instead. The refusal names the format
it refused and the formats this room accepts, so the repair is available rather than
merely implied.

## What a subscriber that cannot decode a publisher's format gets

Nothing. There is no fallback, because a fallback is conversion.

Under a no-conversion rule the repair sits upstream of this service: the publishing
client is asked for something the room can use. That is stated here as the answer
rather than left for every caller to rediscover.

Where it is enforced is the room's negotiated format set. A room agrees a set at the
point participants join, a publisher may only publish inside it, and a subscriber
that cannot decode the set is told so at join rather than after the room has gone
quiet. The API contract in issue #43 carries the set outward, and the diagnostic in
issue #62 is what names this layer when media does not flow.

The honest cost is that a room is only as capable as its least capable participant,
and the participant who has to change is not always the one who noticed.

## What would reopen this

Entry 8 of issue #1, and nothing else on this board.

Every participant can be asked to send something the room can use, except a caller
on a telephone line, who cannot. That is the one request this position cannot answer
on its own, and it belongs to the maintainer. This record neither answers it nor
assumes an answer. If it is answered in a way that puts such a caller in scope
inside this service, the first paragraph of this record stops being true and the
resource budget is re-measured against whatever replaced it.

## The patent position, in operator terms

Some video formats carry patent claims that reach implementations. An operator
running this on their own machine deserves to know what this software does and does
not do with them.

What this software does: it moves packets it does not decode, and it reads the
headers described above. It contains no encoder and no decoder for any format on the
list. It ships no codec implementation and links none.

What it does not do: it does not decide what an operator's obligations are. That
depends on where they are, what they are doing, and what agreements they already
hold, and this record stops short of telling them.

The formats on the list are not equal in this respect, and the difference is stated
rather than averaged. Opus, VP8, VP9 and AV1 are published with royalty-free
positions from their originators. H.264 is the one on the list with a patent pool
that licenses it, and the pool's terms are about encoding, decoding and distribution
rather than about moving packets.

Issue #106 carries this into the operator's documentation alongside the third-party
notices and the bill of materials, and issue #107 is where the uses this project is
not for are written. This record is the source for both and neither restates it.

## The board at the commit this record lands on

The two searches issue #118 owes, run at this commit:

    gh issue list --repo iderex/relais --state all --search 'codec in:body,title' --json number --jq 'length'
    1
    gh issue list --repo iderex/relais --state all --search 'transcode in:body,title' --json number --jq 'length'
    1

Each returns one rather than the zero the issue recorded when it was written, and
the one is the same issue in both cases:

    gh issue list --repo iderex/relais --state all --search 'codec in:body,title' --json number,title,state --jq '.[] | "#\(.number) [\(.state)] \(.title)"'
    #118 [OPEN] Decide the media formats this project carries, and whether it ever converts one

The count moved because the issue that asks the question uses the words it searches
for, which is the same reason it will move again as soon as this record lands. A
later reader should read the matches rather than the number.

What the zero meant when it was recorded still holds: no issue on this board asks
this project to convert media, and none of them assumed a format list. That is what
made this a hole under the forwarding milestone rather than a detail inside it.
