# 07 — Marketplace & community

## 7.1 Goals

- Give farmers a place to **sell surplus** directly, with price transparency.
- Give buyers (households, restaurants, small retailers) a way to **discover local produce**.
- Give farmers a **peer space** — posts, comments, questions answered by the community and by agronomists.
- Keep moderation light but safe; trust comes from verified profiles and localized reputation, not a heavy-handed algorithm.

## 7.2 Personas in the marketplace layer

- **Seller** — any registered user; elevated to `verified_farmer` after KYC (NIC photo + optional farm geo-tag).
- **Buyer** — any registered user.
- **Agronomist** — can post advisories, pin threads, answer questions with a badge.
- **Moderator** — staff or trusted community members with takedown powers.

## 7.3 Marketplace features

### Listings
- Fields: title, crop (tag from taxonomy), variety (optional), quantity + unit (kg, bunches, dozens, sacks), price per unit, available-from / available-until, pickup / delivery options, photos (up to 6), description, plot/GN division location (fuzzed for privacy; precise shared on buyer match), organic flag, certification (if any).
- Derived: freshness window (auto-close when `available_until` passes), reputation of seller.
- Moderation pipeline: virus/NSFW scan → auto-publish if seller is verified + has prior good listings, else queued.

### Discovery
- List + map view, same filters as crop explorer (category, crop, district).
- Default sort: distance to buyer + recency. Boost verified sellers, penalize stale.
- Saved searches with push alerts ("new rambutan listings in Kandy district").

### Transactions (phased)
- **Phase 4 (launch)** — listing + in-app chat + call button. No payment handling. Cash on pickup.
- **Phase 5** — integrated payment via **PayHere** (local SL gateway) + **escrow** held until buyer confirms pickup/delivery. Disputes routed to support.
- **Phase 6** — optional logistics partner integration (local delivery services).

### Reviews
- After a transaction (marked complete by both parties or timed out), both sides can rate 1–5 + leave a short review. Reviews are tied to the transaction, not a public free-text wall.

### Anti-fraud
- Rate-limit listings per user per day.
- Detect duplicate-image listings (perceptual hashing) — allow, but flag.
- Price sanity checks against HARTI reference prices; listings far from the band get a "price looks unusual" tag.
- Shadow-ban patterns: listings with phone numbers in the description (we want communication to flow through the app for safety and dispute resolution).

## 7.4 Community / social features

### Feed
- Chronological + "For you" tabs. "For you" uses simple signals: same district, same crops as the user's plots, followed users, agronomist picks.
- Post types: photo, text, question (tagged with crop/disease), poll, short video (≤ 60 s).
- Reactions: like, "helpful", "same problem here". The "helpful" count surfaces good answers on questions.

### Questions
- A post can be marked `type=question`. Answers can be upvoted. The asker can mark one answer as "solution."
- Agronomist answers carry a verified badge and surface higher.
- Unanswered questions ≥ 48 h are routed to the agronomist on-call.

### Groups (phase 6)
- District groups auto-seeded (one per district). Crop-specific groups (Tea growers, Home gardeners). Opt-in.

### Following
- Follow users, crops, districts, and diseases. Notifications respect a quiet-hours setting.

### Messaging
- 1:1 chat between buyer/seller or any two users (privacy controls: only verified users can DM by default).
- WebSocket-backed. Offline messages delivered on next connect.
- Report + block from any conversation.

## 7.5 Moderation

- **Automated** — NSFW filter, spam classifier, profanity list (SI/TA/EN with local dialect variants), link restrictions.
- **Community** — report button everywhere. Three reports from distinct users ⇒ hidden pending review.
- **Human** — moderator dashboard with queue, evidence, one-click actions (warn, hide, suspend, ban). All actions logged and reversible.
- **Policy** — clearly published in-app (SI/TA/EN). Prohibited: scams, banned agrochemicals, non-crop items, harassment, political campaigning, misinformation on crop safety.

## 7.6 Reputation

Score per user (0–100) computed from:
- Verification level.
- Successful transactions.
- Review average + count.
- Community helpful-count.
- Time on platform.
- Negative signals: reports upheld, disputes lost, shadow-ban triggers.

Surfaced as a badge, not a raw number. Protects new users from instant distrust while still rewarding track record.

## 7.7 Data model additions (extends doc 03)

```
listing(id, seller_id, crop_id, variety_id, quantity, unit,
        price_per_unit, currency, available_from, available_until,
        description_i18n, geom, district_id, organic bool,
        status, created_at, sold_at)

listing_image(id, listing_id, url, order_idx, phash)

listing_contact(id, listing_id, buyer_id, created_at, method)   # expressed interest

transaction(id, listing_id, buyer_id, seller_id,
            quantity, total_price, status, payment_ref nullable,
            created_at, completed_at)

review(id, transaction_id, rater_id, ratee_id, stars, body, created_at)

post(id, author_id, type, body_i18n, media_refs int[],
     crop_id nullable, disease_id nullable, district_id nullable,
     status, created_at)

comment(id, post_id, parent_comment_id nullable, author_id, body_i18n, created_at)

reaction(id, target_type, target_id, user_id, kind, created_at)

follow(follower_id, subject_type, subject_id, created_at)

message_thread(id, participant_ids int[])
message(id, thread_id, sender_id, body, media_refs int[], created_at, read_at)

report(id, target_type, target_id, reporter_id, reason, body, status, created_at)
```

## 7.8 Phasing

- Phase 4 MVP: listings + in-app chat + basic feed + questions.
- Phase 5: payments + reviews + reputation.
- Phase 6: groups, logistics, monetization (featured listings for a fee; never pay-to-win in organic feed).
