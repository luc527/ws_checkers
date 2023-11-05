import sys
import re

passed = True
packages_below = []

for line in sys.stdin:
    line = line.rstrip()

    match_cov = re.search(r'coverage: (\d+\.\d+)%', line)
    if match_cov is None:
        continue

    percentage = float(match_cov.group(1))

    if percentage < 70:
        passed = False

print()
if not passed:
    print('Quality gate failed')
    exit(1)
else:
    print('Quality gate passed')
    exit(0)
