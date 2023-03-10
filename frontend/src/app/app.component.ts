import { Component } from '@angular/core';
import { StoreService } from './services/store-service.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  constructor(public store : StoreService){}
}
