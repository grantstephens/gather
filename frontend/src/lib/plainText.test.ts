import { describe, expect, it } from 'vitest'
import { blurbToPlainText } from './plainText'

const sample = `Birnam Book Festival turns five this weekend, with poetry and a pastry on Saturday and a Hogmanay ghost story on Sunday.

### Poetry Brunch, Birnam Arts, Birnam

Saturday morning at Birnam Arts, you get a free cuppa and a pastry. [Poetry Brunch](/event/poetry-brunch-l9lbpw21) is led by Perthshire poet Morag Anderson.

**Also on:** [Swan Lake](/event/swan-lake-hmruxf7x) is in Pitlochry on Sunday night.`

describe('blurbToPlainText', () => {
  it('matches the server-side description for the sample blurb', () => {
    // Keep in sync with TestStripMarkdownInline / TestBuildPicksHTML_LongBlurb in internal/seo
    expect(blurbToPlainText(sample)).toBe(
      'Birnam Book Festival turns five this weekend, with poetry and a pastry on Saturday and a Hogmanay ghost story on Sunday. Poetry Brunch, Birnam Arts, Birnam...'
    )
  })
  it('spaces block boundaries', () => {
    expect(blurbToPlainText('## Head\n\nBody')).toBe('Head Body')
  })
  it('decodes entities and handles empty', () => {
    expect(blurbToPlainText('Fish & chips')).toBe('Fish & chips')
    expect(blurbToPlainText('')).toBe('')
  })
})
