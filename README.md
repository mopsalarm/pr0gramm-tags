# Tag-Suche für pr0gramm

## Userscript

Das Userscript kann über diesen Link installiert werden:
https://github.com/mopsalarm/pr0gramm-tags/raw/master/pr0gramm-tags.user.js (benötigt [tampermonkey für Chrome](https://chrome.google.com/webstore/detail/tampermonkey/dhdgffkkebhmkfjojejmpbldmpobfkfo) oder [greasemonkey für Firefox](https://addons.mozilla.org/en-US/firefox/addon/greasemonkey/))

**ACHTUNG: Aktuell muss einr Suche ein Fragezeichen vorgestellt werden, damit
sie über das Userscript verarbeiet wird. Beispielsweise: `? webm -sound|gif`**

## Syntax

Es gibt drei Operatoren, um Tags zu einer Suchanfrage zu kombinieren.
* **und**: Möchte man nur Suchergebnisse haben, in denen von zwei Ausdrücken beide vorkommen sollen,
  können diese mit dem Zeichen `&` verknüpft werden - alternativ funktioniert auch das Wörtchen `and`.
  Um Suchanfragen zu vereinfachen, kann das `&`-Zeichen meistens weggelassen werden. 
  
  Beispiel: `facebook & 9gag`, `facebook and 9gag` sowie `facebook 9gag` findet beides Posts,
  die sowohl den Tag `facebook` als auch den Tag `9gag` enthalten.
  
* **oder**: Zwei Ausdrücke können mit dem Zeichen `|` verknüpft werden, um eine Oder-Beziehung herzustellen.

  Beispielsweise: `facebook | 9gag` oder `kadsen or kefer`

* **ohne**: Um alle Posts zu finden, die auf eine Suchanfrage passen, aber eine andere ausschließen, kann das `-` verwendet werden.
  Der erste Tag kann dabei weggelassen werden, so dass alle Posts gefunden werden, die einen bestimmten Tag nicht haben.
 
  Einfache Beispiele sind: `webm -sound` (Alle Videos ohne Ton), `-8015-süßvieh`

Es gibt einge spezielle Suchwörter:
* `u:username` Findet Posts des angegeben Benutzers. Für guten Content z.B. `u:mopsalarm`.
* `f:text` Findet Posts, auf denen Text erkannt wurde.
* `f:top` Findet Posts in top.
* `f:controversial` Posts, die als kontrovers eingestuft werden können. Das bedeutet relativ viele Up und Downvotes in einem ausgewogenen Verhältnis.
* `f:sfw`, `f:nsfw`, `f:nsfl`, `f:nsfp` Findet Posts mit den entsprechend gesetzten Filtern.
* `f:sound` Findet nur Posts, die auch wirklich das Audio-Flag gesetzt haben.
* `f:repost` Findet nur Posts, die mit `repost` getaggt sind. Das Wort muss dabei alleine in einem einzelnen Tag vorkommen, nicht etwa in Kombination wie `kein repost`.
* `s:500`, `s:1000`, `s:1500`, ... Findet Posts, die eine bestimmte mindeste Beniszahl erreicht haben müssen. Heißt konkreter: `s:1000` zeigt nur Posts an, die mindestenes einen Benis von 1000 besitzen.
* `s:shit` Für den wirklich schlechten Content mit Benis kleiner als 300.
* `q:hd`, `q:4k`, `q:1080p`, `q:480p`, `q:kartoffel`, `q:sd` filtert nach verschiedenen Qualitätsstufen.
* `m:ftb`, `m:newfag` für Content von Fliesentischbesitzern und Newfags.

Außerdem kann nach Datum gesucht werden: 
* `d:2014` Findet nur Posts aus 2014.
* `d:2014:04` Findet nur Posts aus dem April 2014.

Wie in der Mathematik gilt hier Punkt-vor-Strich, wobei die Verundung stärker bindet als die Veroderung, und diese wiederum stärker bindet, als das Minus. Es können Klammern gesetzt werden.

Weitere Beispiele:
* `kadse|kefer-0815` Findet alle Posts mit dem Tag `kadse` und alle Posts mit dem Tag `kefer`. Es werden jedoch alle Posts mit dem Tag `0815` aus den Ergebnissen entfernt.
* `-f:nsfl & original content & (f:sfw or (f:nsfw - u:nixname))` Alle posts die original content sind, jedoch kein NSFL, und NSFW nur dann, wenn es nicht von *nixname* ist.
* `s:2000 | (s:500 & (oc | original content))` Alle Posts mit mindestens 2000 Benis sowie OC ab 500 Benis.

## Warnung
Das alles ist Beta und ich gebe keine Garantie, das alles fehlerfrei läuft. Bei Problemen gerne Bescheid sagen.



