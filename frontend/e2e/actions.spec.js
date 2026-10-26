import {test,expect} from '@playwright/test';
import {waitForAppReady,resetMockState,setupConsoleCollector} from './helpers.js';

test.describe('Actions 2.0',()=>{
 let consoleCollector;
 test.beforeEach(async({page})=>{consoleCollector=setupConsoleCollector(page);await resetMockState(page);await page.goto('/');await waitForAppReady(page);});
 test.afterEach(()=>consoleCollector.assertNoErrors());
 const view=page=>page.locator('[data-actions-view]');
 async function openGlobal(page){await page.locator('.sidebar .plugin-item').filter({hasText:'Actions'}).click();await expect(view(page)).toBeVisible();}
 async function add(page,{title='Test application',target='printf',args=['%s','two words; $(literal)'],type='application',scope}={}){
  await view(page).locator('[data-action="add"]').click();
  await page.locator('[data-action-field="title"]').fill(title);
  await page.locator('[data-action-field="type"]').selectOption(type);
  if(type==='application')await expect(page.locator('[data-action-field="interpreter"]')).toBeHidden();
  if(type==='url')await expect(page.locator('[data-action-field="args"]')).toBeHidden();
  await page.locator('[data-action-field="target"]').fill(target);
  if(type!=='url')await page.locator('[data-action-field="args"]').fill(args.map(arg=>JSON.stringify(arg)).join(' '));
  if(scope!==undefined)await page.locator('[data-action-field="scope"]').selectOption(scope);
  if(type==='script')await page.locator('[data-action-field="interpreter"]').fill('python3');
  await page.locator('[data-action="save"]').click();
  await expect(page.locator('[data-action-form]')).toBeHidden();
 }
 async function tab(page){
  for(let n=0;n<8;n++){
   if(await page.getByRole('tab',{name:'Actions',exact:true}).isVisible())break;
   const next=page.locator('[data-workspace-tab-page-next]');if(!await next.isVisible())break;await next.click();
  }
  await page.getByRole('tab',{name:'Actions',exact:true}).click();await expect(view(page)).toBeVisible();
 }
 test('CRUD, order, remount and invalid form recovery',async({page})=>{
  await openGlobal(page);await add(page);await add(page,{title:'Second'});
  await view(page).locator('.actions-card').last().hover();await view(page).locator('[data-action="up"]').last().click();
  await expect(view(page).locator('[data-action="run"]').first()).toHaveText('Second');
  await view(page).locator('.actions-card').first().hover();await view(page).locator('[data-action="edit"]').first().click();
  await page.locator('[data-action-field="title"]').fill('Edited');
  await page.locator('[data-action-field="args"]').fill('"unclosed argument');
  await page.locator('[data-action="save"]').click();await expect(page.locator('[data-action-status]')).toContainText('Close the argument quotes');
  await expect(page.locator('[data-action-field="title"]')).toHaveValue('Edited');
  await page.locator('[data-action-field="args"]').fill('"literal argument"');await page.locator('[data-action="save"]').click();
  await page.locator('.wt-label').filter({hasText:'Project'}).first().click();await openGlobal(page);
  await expect(view(page).locator('[data-action="run"]').first()).toHaveText('Edited');
  await view(page).locator('.actions-card').first().hover();await view(page).locator('[data-action="delete"]').first().click();await view(page).locator('[data-action="confirm-delete"]').click();
  await expect(view(page).locator('[data-action="run"]')).toHaveCount(1);
 });
 test('global and UUID Deal actions stay in scope, including restricted recipes',async({page})=>{
  await openGlobal(page);await add(page,{title:'Global'});
  await page.locator('.wt-label').filter({hasText:'Project'}).first().click();await tab(page);
  await add(page,{title:'Local to Project'});await expect(view(page).locator('[data-action="run"]')).toHaveCount(2);
  await openGlobal(page);await expect(view(page).locator('[data-action="run"]')).toHaveCount(1);
  const [saved]=await page.evaluate(()=>window.go.api.App.ReadPluginSetting('verstak.actions','actions'));
  expect(saved.find(r=>r.title==='Local to Project').workspaceId).toMatch(/^[0-9a-f-]{36}$/);
  await page.locator('button[title="New Deal"]').click();await page.locator('[data-new-deal-name]').fill('Another Deal');await page.locator('[data-new-deal-create]').click();await tab(page);
  await expect(view(page).locator('[data-action="run"]')).toHaveCount(1);await expect(view(page)).not.toContainText('Local to Project');
 });
 test('review decline, one-click launch, edited arguments and permission revoke',async({page})=>{
  await openGlobal(page);await add(page);
  await page.locator('[data-action="run"]').click();await expect(page.locator('[data-action-status]')).toContainText('nothing was run');
  expect(await page.evaluate(()=>window.__actionsMockRuns || [])).toHaveLength(0);
  await page.evaluate(()=>window.__actionsMockAllowReview=true);
  await page.locator('[data-action="run"]').click();await expect(page.locator('[data-action-output]')).toContainText('two words; $(literal)');
  const reviews=await page.evaluate(()=>window.__actionsMockReviews);
  await page.locator('[data-action="run"]').click();await expect.poll(()=>page.evaluate(()=>window.__actionsMockRuns.length)).toBe(2);
  expect(await page.evaluate(()=>window.__actionsMockReviews)).toBe(reviews);
  await page.locator('.actions-card').hover();await page.locator('[data-action="edit"]').click();await page.locator('[data-action-field="args"]').fill('changed');await page.locator('[data-action="save"]').click();await page.locator('[data-action="run"]').click();
  await expect.poll(()=>page.evaluate(()=>window.__actionsMockReviews)).toBe(reviews+1);await expect(page.locator('[data-action-output]')).toContainText('changed');
  await page.locator('.actions-card').hover();await page.locator('[data-action="forget"]').click();await expect(page.locator('[data-action-status]')).toContainText('revoked');
  expect(await page.evaluate(()=>Object.keys(window.__actionsMockGrants))).toHaveLength(1);
 });
 test('absent program gives a recovery message',async({page})=>{
  await openGlobal(page);await add(page,{target:'missing-verstak-program'});await page.evaluate(()=>window.__actionsMockAllowReview=true);
  await page.locator('[data-action="run"]').click();await expect(page.getByRole('alert')).toContainText('Program not found');
 });
 test('sync arrival and reopening never launch or approve imported definitions',async({page})=>{
  await openGlobal(page);
  await page.evaluate(async()=>{
   await window.go.api.App.WritePluginSetting('verstak.actions','actions',[{id:'imported',title:'Imported',icon:'external-link',type:'application',target:'printf',args:['literal'],cwd:'',mode:'wait',workspaceId:'',rank:1,approved:true}]);
   window.__VERSTAK_DISPATCH_BACKEND_EVENT__({name:'plugin.records.changed',payload:{pluginId:'verstak.actions',document:'actions'}});
  });
  await expect(page.locator('[data-action="run"]')).toHaveText('Imported');
  await page.evaluate(async()=>{
    const [records]=await window.go.api.App.ReadPluginSetting('verstak.actions','actions');
    await window.go.api.App.WritePluginSetting('verstak.actions','actions',[...records,{...records[0],id:'imported-2',title:'Imported 2'}]);
    window.__VERSTAK_DISPATCH_BACKEND_EVENT__({name:'plugin.records.changed',payload:{pluginId:'verstak.actions',document:'actions'}});
  });
  await expect(page.locator('[data-action="run"]')).toHaveCount(2);
  await page.locator('.actions-card').last().hover();await page.locator('[data-action="up"]').last().click();await expect(page.locator('[data-action="run"]').first()).toHaveText('Imported 2');
  expect(await page.evaluate(()=>window.__actionsMockRuns || [])).toHaveLength(0);expect(await page.evaluate(()=>window.__actionsMockReviews || 0)).toBe(0);
  await page.locator('[data-action="run"]').first().click();await expect(page.locator('[data-action-status]')).toContainText('nothing was run');
 });
 test('script interpreter and URL remain separate structured requests',async({page})=>{
  await openGlobal(page);await page.evaluate(()=>window.__actionsMockAllowReview=true);
  await add(page,{title:'Script',type:'script',target:'/tmp/test script.py',args:['two words']});await page.locator('[data-action="run"]').click();await expect(page.locator('[data-action-status]')).toContainText('exit code: 0');
  expect(await page.evaluate(()=>window.__actionsMockRuns[0].request)).toMatchObject({executable:'python3',args:['/tmp/test script.py','two words']});
  await add(page,{title:'Link',type:'url',target:'https://example.org/test?q=one%20two'});await page.locator('[data-action="run"]').filter({hasText:'Link'}).click();await expect.poll(()=>page.evaluate(()=>window.__actionsMockRuns.length)).toBe(2);
  expect(await page.evaluate(()=>window.__actionsMockRuns[1].request)).toEqual({kind:'url',url:'https://example.org/test?q=one%20two'});
 });
 test('stop cancels a pending process and disable preserves definitions',async({page})=>{
  await openGlobal(page);await add(page);await page.evaluate(()=>{
   window.__actionsMockAllowReview=true;
   window.go.api.App.PluginProcessRun=(_plugin,id)=>new Promise(resolve=>{window.__actionsPending={id,resolve};});
   window.go.api.App.PluginProcessCancel=async(_plugin,id)=>{if(window.__actionsPending?.id===id)window.__actionsPending.resolve({started:true,exitCode:-1,cancelled:true,stdout:'',stderr:''});};
  });
  await page.locator('[data-action="run"]').click();await page.locator('[data-action="stop"]').click();await expect(page.locator('[data-action-status]')).toContainText('Process stopped');
  await page.evaluate(async()=>{await window.go.api.App.DisablePlugin('verstak.actions');window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));});
  await expect(page.locator('.sidebar .plugin-item').filter({hasText:'Actions'})).toHaveCount(0);
  await page.evaluate(async()=>{await window.go.api.App.EnablePlugin('verstak.actions');window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));});await openGlobal(page);await expect(page.locator('[data-action="run"]')).toHaveText('Test application');
 });
 test('command palette opens global actions',async({page})=>{
  await page.keyboard.press('Control+Shift+P');const input=page.getByRole('dialog').getByRole('textbox');await expect(input).toBeVisible();await input.fill('Actions: open global');await page.keyboard.press('Enter');await expect(view(page)).toBeVisible();await expect(view(page)).toContainText('Global actions');
 });
 test('RU and EN labels update through the host language controls',async({page})=>{
  await openGlobal(page);await page.locator('[data-settings-menu-button]').click();await page.locator('[data-settings-language="ru"]').click();
  await page.getByRole('button',{name:'Закрыть',exact:true}).click();
  await expect(view(page).locator('[data-action="add"]')).toHaveText('Добавить действие');
  await expect(view(page)).toContainText('Глобальные действия');
  await page.locator('[data-settings-menu-button]').click();await page.locator('[data-settings-language="en"]').click();
  await page.getByRole('button',{name:'Close',exact:true}).click();
  await expect(view(page).locator('[data-action="add"]')).toHaveText('Add action');
 });
 test('built-in icons and narrow layout in both themes',async({page})=>{
  await openGlobal(page);await add(page);await expect(page.locator('[data-action="run"] svg[data-icon="external-link"]')).toHaveCount(1);await page.setViewportSize({width:640,height:760});
  for(const theme of ['light','dark']){
   await page.evaluate(async(theme)=>{await window.go.api.App.UpdateAppSettings({theme});document.documentElement.setAttribute('data-theme',theme);},theme);
   await expect(page.locator('[data-action="run"]')).toBeVisible();expect(await view(page).evaluate(node=>node.scrollWidth<=node.clientWidth+1)).toBe(true);
  }
  await page.screenshot({path:'e2e-results/actions-narrow.png'});
 });
 test('argument line groups quotes, preserves paths and never expands shell symbols',async({page})=>{
  await openGlobal(page);await add(page);
  await page.locator('.actions-card').hover();await page.locator('[data-action="edit"]').click();
  const line=String.raw`connect "my alias" '' C:\tools\bin "\\server\team folder" --title="two words" 'single words' "say \"hi\"" $HOME *.txt '$(touch file)'`;
  await page.locator('[data-action-field="args"]').fill(line);
  await page.locator('[data-action="save"]').click();await expect(page.locator('[data-action-form]')).toBeHidden();
  const expected=['connect','my alias','',String.raw`C:\tools\bin`,String.raw`\\server\team folder`,'--title=two words','single words','say "hi"','$HOME','*.txt','$(touch file)'];
  const [records]=await page.evaluate(()=>window.go.api.App.ReadPluginSetting('verstak.actions','actions'));
  expect(records[0].args).toEqual(expected);
  await page.evaluate(()=>window.__actionsMockAllowReview=true);await page.locator('[data-action="run"]').click();
  await expect.poll(()=>page.evaluate(()=>window.__actionsMockRuns?.length||0)).toBe(1);
  expect(await page.evaluate(()=>window.__actionsMockRuns[0].request.args)).toEqual(expected);
  // Editing and adding a new argument exercises the formatter as well as the parser.
  await page.locator('.actions-card').hover();await page.locator('[data-action="edit"]').click();
  const input=page.locator('[data-action-field="args"]');await input.fill((await input.inputValue())+' tail');
  await page.locator('[data-action="save"]').click();await expect(page.locator('[data-action-form]')).toBeHidden();
  expect((await page.evaluate(()=>window.go.api.App.ReadPluginSetting('verstak.actions','actions')))[0][0].args).toEqual([...expected,'tail']);
 });
 test('existing argv survives title-only edits without reparsing',async({page})=>{
  await openGlobal(page);await add(page);
  const oldArgs=['', 'a"b', 'line\nbreak', 'two words', String.raw`C:\directory with spaces`+String.fromCharCode(92)];
  await page.evaluate(async(args)=>{const [records]=await window.go.api.App.ReadPluginSetting('verstak.actions','actions');records[0].args=args;await window.go.api.App.WritePluginSetting('verstak.actions','actions',records);window.__VERSTAK_DISPATCH_BACKEND_EVENT__({name:'plugin.records.changed',payload:{pluginId:'verstak.actions',document:'actions'}});},oldArgs);
  await page.locator('.actions-card').hover();await page.locator('[data-action="edit"]').click();await page.locator('[data-action-field="title"]').fill('Renamed');await page.locator('[data-action="save"]').click();await expect(page.locator('[data-action-form]')).toBeHidden();
  expect((await page.evaluate(()=>window.go.api.App.ReadPluginSetting('verstak.actions','actions')))[0][0].args).toEqual(oldArgs);
 });
 test('shared icon modal selects sprite icons, cancels and preserves form input',async({page})=>{
  await openGlobal(page);await page.locator('[data-action="add"]').click();
  await page.locator('[data-action-field="title"]').fill('Rocket');await page.locator('[data-action-field="target"]').fill('printf');
  await page.locator('[data-action-field="args"]').fill('"two words"');
  await page.locator('[data-action="pick-icon"]').click();
  const dialog=page.getByRole('dialog',{name:'Choose icon'});await expect(dialog).toBeVisible();
  await dialog.getByRole('searchbox').fill('rocket');
  const choice=dialog.getByRole('button',{name:'rocket',exact:true});await expect(choice).toBeVisible();
  await expect(choice.locator('svg path')).not.toHaveCount(0);
  const glyph=await choice.locator('svg').evaluate(svg=>[...svg.children].map(n=>n.outerHTML).join(''));await choice.click();await expect(dialog).toBeHidden();
  await expect(page.locator('[data-action-field="args"]')).toHaveValue('"two words"');
  await expect(page.locator('[data-action="pick-icon"]')).toBeFocused();
  await page.locator('[data-action="pick-icon"]').click();await dialog.getByRole('searchbox').fill('banana');await page.keyboard.press('Escape');await expect(dialog).toBeHidden();
  await expect(page.locator('[data-action="pick-icon"]')).toContainText('rocket');
  await page.locator('[data-action="save"]').click();await expect(page.locator('[data-action-form]')).toBeHidden();
  const icon=page.locator('[data-action="run"] svg[data-icon="rocket"]');await expect(icon).toBeVisible();
  expect(await icon.evaluate(svg=>[...svg.children].map(n=>n.outerHTML).join(''))).toBe(glyph);
  await page.locator('.actions-card').hover();await page.locator('[data-action="edit"]').click();await expect(page.locator('[data-action="pick-icon"]')).toContainText('rocket');
  await page.locator('[data-action="pick-icon"]').click();await expect(dialog).toBeVisible();
  await page.keyboard.press('Escape');await expect(dialog).toBeHidden();
  await page.evaluate(()=>{window.__pickerAPI=window.createPluginAPI('verstak.actions');window.__pickerResult='pending';window.__pickerAPI.ui.pickIcon('rocket').then(value=>window.__pickerResult=value);});
  await expect(dialog).toBeVisible();await page.evaluate(()=>window.__pickerAPI.dispose());await expect(dialog).toBeHidden();
  await expect.poll(()=>page.evaluate(()=>window.__pickerResult)).toBe(null);
 });
 test('24 compact cards fit, controls appear on hover and keyboard focus without launching',async({page})=>{
  await openGlobal(page);await add(page);
  await page.evaluate(async()=>{
   const [records]=await window.go.api.App.ReadPluginSetting('verstak.actions','actions');
   await window.go.api.App.WritePluginSetting('verstak.actions','actions',Array.from({length:24},(_,n)=>({...records[0],id:'dense-'+n,title:'Application '+(n+1),description:'A tooltip description',rank:n*1024})));
   window.__VERSTAK_DISPATCH_BACKEND_EVENT__({name:'plugin.records.changed',payload:{pluginId:'verstak.actions',document:'actions'}});
  });
  await expect(page.locator('.actions-card')).toHaveCount(24);
  await page.setViewportSize({width:1280,height:800});await page.mouse.move(0,0);
  const card=page.locator('.actions-card').first(),controls=card.locator('.actions-card-controls');
  await expect(controls).toHaveCSS('opacity','0');
  expect(await view(page).evaluate(node=>node.scrollHeight<=node.clientHeight+1)).toBe(true);
  await card.hover();await expect(controls).toHaveCSS('opacity','1');
  await page.mouse.move(0,0);await card.locator('[data-action="run"]').focus();await expect(controls).toHaveCSS('opacity','1');
  await page.keyboard.press('Tab');await expect(card.locator('[data-action="edit"]')).toBeFocused();await page.keyboard.press('Enter');await expect(page.locator('[data-action-form]')).toBeVisible();
  expect(await page.evaluate(()=>window.__actionsMockRuns||[])).toHaveLength(0);
  await page.locator('[data-action="cancel"]').click();await page.mouse.move(0,0);
  await expect(view(page)).not.toContainText('Definitions sync');
  await page.screenshot({path:'e2e-results/actions-compact.png'});
 });

});
