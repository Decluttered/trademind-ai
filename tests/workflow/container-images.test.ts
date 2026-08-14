import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const root = path.resolve(__dirname, '..', '..');
const workflow = readFileSync(path.join(root, '.github', 'workflows', 'container-images.yml'), 'utf8');

describe('container image release workflow', () => {
  it('supports branch builds and v-prefixed release tags', () => {
    expect(workflow).toContain('      - main');
    expect(workflow).toContain('      - dev');
    expect(workflow).toContain('    tags:\n      - "v*"');
  });

  it('requires the release tag to match IMAGE_VERSION', () => {
    expect(workflow).toContain('expected_tag="v${version}"');
    expect(workflow).toContain(
      'Release tag $GITHUB_REF_NAME must exactly match deploy/IMAGE_VERSION as $expected_tag',
    );
  });

  it('requires the tagged commit to be contained in main', () => {
    expect(workflow).toContain('git fetch --no-tags origin +refs/heads/main:refs/remotes/origin/main');
    expect(workflow).toContain('git merge-base --is-ancestor "$release_commit" refs/remotes/origin/main');
  });

  it('publishes stable version tags and latest only for releases', () => {
    expect(workflow).toContain(
      "type=raw,value=v${{ needs.image_metadata.outputs.version }},enable=${{ needs.image_metadata.outputs.is_release == 'true' }}",
    );
    expect(workflow).toContain(
      "type=raw,value=${{ needs.image_metadata.outputs.version }},enable=${{ needs.image_metadata.outputs.is_release == 'true' }}",
    );
    expect(workflow).toContain(
      "type=raw,value=latest,enable=${{ needs.image_metadata.outputs.is_release == 'true' }}",
    );
    expect(workflow).not.toContain("needs.image_metadata.outputs.ref_slug == 'main'");
  });

  it('keeps branch and branch-version tags on non-release builds', () => {
    expect(workflow).toContain(
      "type=raw,value=${{ needs.image_metadata.outputs.ref_slug }},enable=${{ needs.image_metadata.outputs.is_release != 'true' }}",
    );
    expect(workflow).toContain(
      "type=raw,value=${{ needs.image_metadata.outputs.ref_slug }}-v${{ needs.image_metadata.outputs.version }},enable=${{ needs.image_metadata.outputs.is_release != 'true' }}",
    );
  });
});
