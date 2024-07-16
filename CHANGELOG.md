# v0.3.1 (2024-07-16)

* Bumped minimum Go version to 1.21.
* Added support for authenticated media.
* Added `/powerlevel` command for managing power levels.
* Disabled logging by default.
* Changed default log directory to `~/.local/state/gomuks` on Linux.

# v0.3.0 (2022-11-19)

* Bumped minimum Go version to 1.18.
* Switched from `/r0` to `/v3` paths everywhere.
  * The new `v3` paths are implemented since Synapse 1.48, Dendrite 0.6.5,
    and Conduit 0.4.0. Servers older than these are no longer supported.
* Added config flags for backspace behavior.
* Added `/rainbownotice` command to send a rainbow as a `m.notice` message.
* Added support for editing messages in an external editor.
* Added arrow key support for navigating results in fuzzy search.
* Added initial support for configurable keyboard shortcuts
  (thanks to [@3nprob] in [#328]).
* Added support for shortcodes *without* tab-completion in `/react`
  (thanks to [@tleb] in [#354]).
* Added background color to differentiate `inline code`
  (thanks to [@n-peugnet] in [#361]).
* Added tab-completion support for `/toggle` options
  (thanks to [@n-peugnet] in [#362]).
* Added initial support for rendering spoilers in messages.
* Added support for sending spoilers (with `||reason|spoiler||` or `||spoiler||`).
* Added support for inline links (limited terminal support; requires
  `/toggle inlineurls`).
* Added graphical file picker for `/upload` when no path is provided
  (requires `zenity`).
* Updated more places to use default/reverse colors instead of white/black to
  better work on light themed terminals (thanks to [@n-peugnet] in [#401]).
* Fixed mentions being lost when editing messages.
* Fixed date change messages showing the wrong date.
* Fixed some whitespace in HTML being rendered even when it shouldn't.
* Fixed copying non-text messages with `/copy`.
* Fixed rendering code blocks with unknown languages
  (thanks to [@n-peugnet] in [#386]).
* Fixed newlines not working in code blocks with certain syntax highlightings
  (thanks to [@n-peugnet] in [#387]).
* Fixed rendering more than one reaction of the same type in a single message
  (thanks to [@n-peugnet] in [#391]).
* Fixed line-wrapped messages getting corrupted when receiving a reaction
  (thanks to [@n-peugnet] in [#397]).

[@3nprob]: https://github.com/3nprob
[@tleb]: https://github.com/tleb
[@n-peugnet]: https://github.com/n-peugnet
[#328]: https://github.com/tulir/gomuks/pull/328
[#354]: https://github.com/tulir/gomuks/pull/354
[#361]: https://github.com/tulir/gomuks/pull/361
[#362]: https://github.com/tulir/gomuks/pull/362
[#401]: https://github.com/tulir/gomuks/pull/401

# v0.2.4 (2021-09-21)

* Added `is_direct` flag when creating DMs (thanks to [@gsauthof] in [#261]).
* Added `newline` toggle for swapping enter and alt-enter behavior
  (thanks to [@octeep] in [#270]).
* Added `timestamps` toggle for disabling timestamps in the UI
  (thanks to [@lxea] in [#304]).
* Added support for getting custom download directory with `xdg-user-dir`.
* Added support for updating homeserver URL based on well-known data in
  `/login` response.
* Updated some places to use default color instead of white to better work on
  light themed terminals (thanks to [@zavok] in [#280]).
* Updated notification library to work on all unix-like systems with `notify-send`.
    * Notification sounds will now work if either `paplay` or `ogg123` is available.
    * Based on work by [@negatethis] (in [#298]) and [@begss] (in [#312]).
* Disabled logging request content for sensitive requests like `/login` and
  cross-signing key uploads.
* Fixed caching state of rooms where the room ID contains slashes.
* Fixed index error in fuzzy search (thanks to [@Evidlo] in [#268]).

[@gsauthof]: https://github.com/gsauthof
[@octeep]: https://github.com/octeep
[@lxea]: https://github.com/lxea
[@zavok]: https://github.com/zavok
[@negatethis]: https://github.com/negatethis
[@begss]: https://github.com/begss
[@Evidlo]: https://github.com/Evidlo
[#261]: https://github.com/tulir/gomuks/pull/261
[#268]: https://github.com/tulir/gomuks/pull/268
[#270]: https://github.com/tulir/gomuks/pull/270
[#280]: https://github.com/tulir/gomuks/pull/280
[#298]: https://github.com/tulir/gomuks/pull/298
[#304]: https://github.com/tulir/gomuks/pull/304
[#312]: https://github.com/tulir/gomuks/pull/312

# v0.2.3 (2021-02-19)

* Switched crypto store to use SQLite to prevent it from getting corrupted all
  the time.
* Added macOS builds (both x86 and arm64).
* Allowed password login to servers with both SSO and password login enabled.

# v0.2.2 (2021-01-06)

* Added some initial cross-signing/SSSS commands.
* Updated mautrix-go to fix Go 1.15.3+ compatibility.
* Fixed text selection panic caused by clipboard.
* Fixed incoming encryption state events not being detected.
* Fixed zombie processes left from opening files (thanks to [@Midek] in [#234]).

[@Midek]: https://github.com/Midek
[#234]: https://github.com/tulir/gomuks/pull/234

# v0.2.1 (2020-10-23)

* Moved help into a modal (partially done by [@wvffle] in [#223]).
* Fixed choosing a login flow when logging in.
* Fixed edits by different users than the original message sender being rendered.
* Fixed panic when rendering empty code block.
* Fixed panic in `/open` command (thanks to [@dec05eba] in [#226]).
* Fixed command autocompletion (thanks to [@wvffle] in [#222]).

[@dec05eba]: https://github.com/dec05eba
[#222]: https://github.com/tulir/gomuks/pull/222
[#223]: https://github.com/tulir/gomuks/pull/223
[#226]: https://github.com/tulir/gomuks/pull/226

# v0.2.0 (2020-09-04)

* Added interactive device verification support (only outgoing requests currently).
* Added option to show inline link target as text (thanks to [@r3k2] in [#189]).
* Added `/edit` command as an alternative to <kbd>↑</kbd>/<kbd>↓</kbd>.
* Added support for importing and exporting message decryption keys.
* Added command for uploading files (started by [@wvffle] in [#206]).
* Added parameter autocompletion for some commands (mostly the new crypto and
  upload commands, but also `/download` and `/open`).
* Fixed autocompleting HTML pills when markdown is disabled.
* Fixed editing the same message many times.
* Fixed mangled comment newlines in code blocks (thanks to [@wvffle] in [#214]).

[@wvffle]: https://github.com/wvffle
[@r3k2]: https://github.com/r3k2
[#189]: https://github.com/tulir/gomuks/pull/189
[#206]: https://github.com/tulir/gomuks/pull/206
[#214]: https://github.com/tulir/gomuks/pull/214

# v0.1.2 (2020-06-24)

* Fixed panic when clicking <kbd>Shift</kbd>+<kbd>Tab</kbd> on the first item
  of the fuzzy room search dialog.
* Fixed panic when rendering `m.room.canonical_alias` events with no
  `prev_content`.
* Fixed rendering displayname changes.

# v0.1.1 (2020-06-24)

No changelog available.

# v0.1.0 (2020-05-10)

Initial release.
