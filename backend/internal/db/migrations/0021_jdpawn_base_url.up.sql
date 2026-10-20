-- jdpawn's TLS certificate covers jdpawn.com.tw only, so https://www.jdpawn.com.tw fails
-- verification and every crawl errored. The crawler now uses the bare host.
UPDATE sources SET base_url = 'https://jdpawn.com.tw' WHERE key = 'jdpawn';
