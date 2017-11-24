// ==UserScript==
// @name         pr0gramm-tags
// @namespace    http://github.net/mopsalarm/pr0gramm-tags
// @version      1.0.1
// @description  Allow better searches
// @author       Mopsalarm
// @match        http://pr0gramm.com/*
// @match        https://pr0gramm.com/*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

window.eval(`
  (function () {
      'use strict';

      var orgGet = p.api.get;
      p.api.get = function (endpoint, opts, success, error) {
          if (endpoint === "items.get") {
              var hasSpecialPrefix = (opts.tags || "")[0] === "?";
              if (hasSpecialPrefix) {
                  opts.tags = opts.tags.slice(1);
                  return jQuery.ajax({
                      type: "GET",
                      url: "//app.pr0gramm.com/api/categories/v1/general",
                      success: success,
                      error: error,
                      dataType: "json",
                      data: opts
                  });
              }
          }

          return orgGet.apply(this, arguments);
      };

      // fix reloading issues
      if(/\\/\\?/.test(decodeURIComponent(p.getLocation() || ""))) {
          p.navigateTo(p.getLocation(), p.NAVIGATE.FORCE);
      }

  })();
`);
