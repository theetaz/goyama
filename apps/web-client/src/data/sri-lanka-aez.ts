/**
 * Rough Sri Lankan agro-ecological-zone polygons — hand-authored for the first
 * visual pass of the map UX. These are NOT the NRMC canonical boundaries;
 * they're sketchy bounds sufficient to show the overlay pattern and let a
 * farmer understand "my plot is in the Wet Zone" or similar.
 *
 * Replace with the vectorised NRMC dataset when corpus/seed/aez/ gains real
 * geometry. The canonical schema expects a GeoJSON MultiPolygon per record.
 *
 * Zone colours are read by the map layers via the `zone` property; the actual
 * colour tokens live in packages/design-tokens (aez.wet / .intermediate / .dry).
 */

import type { FeatureCollection, Polygon } from 'geojson';

export interface AezProperties {
  code: 'WZ' | 'IZ' | 'DZ';
  name: string;
  name_si: string;
  name_ta: string;
  group: 'wet' | 'intermediate' | 'dry';
  summary: string;
}

export const sriLankaAez: FeatureCollection<Polygon, AezProperties> = {
  type: 'FeatureCollection',
  features: [
    {
      type: 'Feature',
      properties: {
        code: 'WZ',
        name: 'Wet Zone',
        name_si: 'තෙත් කලාපය',
        name_ta: 'ஈர வலயம்',
        group: 'wet',
        summary:
          'SW quadrant + central hills. >2500 mm annual rainfall. Tea, rubber, cinnamon, wet-zone rice, rambutan, mangosteen.',
      },
      geometry: {
        type: 'Polygon',
        coordinates: [
          [
            [79.82, 6.85],
            [80.0, 7.3],
            [80.35, 7.55],
            [80.55, 7.35],
            [80.75, 7.0],
            [80.85, 6.65],
            [80.75, 6.3],
            [80.55, 6.05],
            [80.25, 5.95],
            [79.95, 6.1],
            [79.82, 6.5],
            [79.82, 6.85],
          ],
        ],
      },
    },
    {
      type: 'Feature',
      properties: {
        code: 'IZ',
        name: 'Intermediate Zone',
        name_si: 'අතරමැදි කලාපය',
        name_ta: 'இடைநிலை வலயம்',
        group: 'intermediate',
        summary:
          'Buffer strip between Wet and Dry. 1750–2500 mm. Mixed cropping, banana, mango, vegetables; rice in both Maha and Yala.',
      },
      geometry: {
        type: 'Polygon',
        coordinates: [
          [
            [80.0, 7.3],
            [80.4, 7.9],
            [80.75, 8.2],
            [81.15, 7.9],
            [81.2, 7.3],
            [81.05, 6.7],
            [80.85, 6.3],
            [80.75, 6.0],
            [80.5, 5.95],
            [80.3, 6.15],
            [80.15, 6.45],
            [80.05, 6.85],
            [80.0, 7.3],
          ],
        ],
      },
    },
    {
      type: 'Feature',
      properties: {
        code: 'DZ',
        name: 'Dry Zone',
        name_si: 'වියළි කලාපය',
        name_ta: 'வறட்சி வலயம்',
        group: 'dry',
        summary:
          'North, North-Central, East, and SE. <1750 mm with a distinct May–Sep dry season. Paddy under tank irrigation, big onion, chilli, pulses, mango, cashew.',
      },
      geometry: {
        type: 'Polygon',
        coordinates: [
          [
            [79.75, 9.85],
            [80.25, 9.75],
            [80.85, 9.45],
            [81.35, 9.1],
            [81.75, 8.55],
            [81.85, 8.0],
            [81.75, 7.2],
            [81.55, 6.55],
            [81.25, 6.05],
            [80.95, 5.95],
            [80.95, 6.3],
            [81.05, 6.7],
            [81.2, 7.3],
            [81.15, 7.9],
            [80.75, 8.2],
            [80.4, 7.9],
            [80.0, 7.3],
            [79.82, 6.85],
            [79.75, 7.5],
            [79.85, 8.3],
            [79.85, 9.0],
            [79.75, 9.85],
          ],
        ],
      },
    },
  ],
};

export const sriLankaBounds: [[number, number], [number, number]] = [
  [79.4, 5.8],
  [82.0, 9.95],
];

export const sriLankaCenter = { longitude: 80.77, latitude: 7.87 };
